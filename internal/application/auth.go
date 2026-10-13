package application

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/abdulmuti12/api-point-go/internal/domain"
)

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
func tokenExpired(v any) bool {
	t, e := time.Parse(time.RFC3339Nano, domain.Text(v))
	if e != nil {
		t, e = time.Parse("2006-01-02 15:04:05", domain.Text(v))
	}
	return e != nil || !t.After(time.Now())
}

func (s *Service) Authenticate(ctx context.Context, raw, kind string) (domain.Principal, error) {
	p, e := s.Tokens.Parse(raw, kind)
	if e != nil {
		return p, domain.Fail("Unauthorized", "Token is invalid or has expired")
	}
	n, e := s.Store.Count(ctx, "point_go_revoked_tokens", domain.Q(domain.Equal("jti", p.JTI)))
	if e != nil {
		return p, e
	}
	if n > 0 {
		return p, domain.Fail("Unauthorized", "Token has been revoked")
	}
	table := "admins"
	if kind == "customer" {
		table = "customers"
	}
	user, e := s.Find(ctx, table, p.ID)
	if e != nil {
		return p, missing(e, "Unauthorized", "User tidak ditemukan")
	}
	p.Name = domain.Text(user["name"])
	return p, nil
}
func (s *Service) Roles(ctx context.Context, id string) ([]domain.Record, error) {
	pivots, e := s.Store.List(ctx, "admin_role", domain.Q(domain.Equal("admin_id", id)))
	if e != nil {
		return nil, e
	}
	return s.Store.List(ctx, "roles", domain.Q(domain.In("id", listIDs(pivots, "role_id"))))
}
func (s *Service) Menus(ctx context.Context, id string) ([]domain.Record, error) {
	roles, e := s.Roles(ctx, id)
	if e != nil {
		return nil, e
	}
	rows, e := s.Store.List(ctx, "role_menu", domain.Q(domain.In("role_id", listIDs(roles, "id"))))
	if e != nil {
		return nil, e
	}
	return s.Store.List(ctx, "menus", domain.Q(domain.In("id", listIDs(rows, "menu_id"))))
}
func (s *Service) Authorize(ctx context.Context, p domain.Principal, menu string) error {
	if menu == "" {
		return nil
	}
	if p.Kind != "admin" {
		return domain.Fail("Unauthorized", "Tidak terautentikasi")
	}
	menus, e := s.Menus(ctx, p.ID)
	if e != nil {
		return e
	}
	for _, m := range menus {
		if strings.EqualFold(domain.Text(m["route"]), menu) {
			return nil
		}
	}
	return domain.Reject(403, domain.Record{"success": false, "message": "Tidak memiliki akses ke menu: " + menu, "status": 403})
}
func (s *Service) loginData(user domain.Record, kind string) (domain.Record, error) {
	token, e := s.Tokens.Issue(domain.Principal{ID: domain.Text(user["id"]), Kind: kind})
	if e != nil {
		return nil, e
	}
	return domain.Record{"id": user["id"], "username": user["name"], "email": user["email"], "token": token}, nil
}
func (s *Service) putToken(ctx context.Context, kind, email string, ttl time.Duration) (string, error) {
	token, e := domain.RandomToken()
	if e != nil {
		return "", e
	}
	key := kind + ":" + email
	e = s.Store.Transaction(ctx, func(db domain.Store) error {
		table := "customers"
		if kind == "reset" {
			table = "admins"
		}
		if _, e := db.Find(ctx, table, domain.Query{Filters: []domain.Filter{domain.Equal("email", email)}, Lock: true}); e != nil {
			return e
		}
		if e := db.Delete(ctx, "point_go_tokens", domain.Q(domain.Equal("key", key))); e != nil {
			return e
		}
		_, e := db.Insert(ctx, "point_go_tokens", domain.Record{"key": key, "hash": hashToken(token), "subject": email, "kind": kind, "expires_at": time.Now().Add(ttl)})
		return e
	})
	return token, e
}
func (s *Service) consumeToken(ctx context.Context, kind, email, token string, fn func(domain.Store) error) error {
	return s.Store.Transaction(ctx, func(db domain.Store) error {
		row, e := db.Find(ctx, "point_go_tokens", domain.Query{Filters: []domain.Filter{domain.Equal("key", kind+":"+email)}, Lock: true})
		if errors.Is(e, domain.ErrNotFound) {
			return domain.Fail("Invalid Token", "Token tidak valid atau kadaluarsa")
		}
		if e != nil {
			return e
		}
		if tokenExpired(row["expires_at"]) || subtle.ConstantTimeCompare([]byte(domain.Text(row["hash"])), []byte(hashToken(token))) != 1 {
			return domain.Fail("Invalid Token", "Token tidak valid atau kadaluarsa")
		}
		if e = fn(db); e != nil {
			return e
		}
		return db.Delete(ctx, "point_go_tokens", domain.Q(domain.Equal("key", row["key"])))
	})
}

func (s *Service) auth(ctx context.Context, c domain.Command) (domain.Result, error) {
	in := c.Input
	kind := c.Principal.Kind
	if c.Table == "admins" {
		kind = "admin"
	}
	if kind == "" {
		kind = "customer"
	}
	table := "customers"
	if kind == "admin" {
		table = "admins"
	}
	switch c.Action {
	case "login":
		user, e := s.Optional(ctx, table, domain.Q(domain.Equal("email", in["email"])))
		if e != nil {
			return domain.Result{}, e
		}
		if user == nil || !s.Passwords.Check(domain.Text(in["password"]), domain.Text(user["password"])) {
			if kind == "admin" {
				return domain.Result{}, domain.Fail("Unauthorized", "Email atau password salah")
			}
			return domain.Result{}, domain.Reject(401, domain.Record{"success": false, "message": "Invalid credentials"})
		}
		if kind == "customer" && domain.Integer(user["verify"]) != 1 {
			token, e := s.putToken(ctx, "setup", domain.Text(user["email"]), time.Hour)
			if e != nil {
				return domain.Result{}, e
			}
			return domain.Result{}, domain.Reject(403, domain.Record{"success": false, "message": "Silakan set password baru untuk mengaktifkan akun Anda.", "needs_password_setup": true, "setup_token": token, "email": user["email"]})
		}
		data, e := s.loginData(user, kind)
		if e != nil {
			return domain.Result{}, e
		}
		if kind == "admin" {
			if e = s.Store.Update(ctx, table, idQuery(user["id"]), domain.Record{"last_login": time.Now().UTC()}); e != nil {
				return domain.Result{}, e
			}
			roles, e := s.Roles(ctx, domain.Text(user["id"]))
			if e != nil {
				return domain.Result{}, e
			}
			menus := []domain.Record{}
			if len(roles) > 0 {
				pivots, e := s.Store.List(ctx, "role_menu", domain.Q(domain.Equal("role_id", roles[0]["id"])))
				if e != nil {
					return domain.Result{}, e
				}
				rows, e := s.Store.List(ctx, "menus", domain.Q(domain.In("id", listIDs(pivots, "menu_id"))))
				if e != nil {
					return domain.Result{}, e
				}
				for _, m := range rows {
					menus = append(menus, domain.Record{"id": m["id"], "name": m["name"], "route": m["url"]})
				}
			}
			delete(data, "username")
			data["name"] = user["name"]
			data["role"] = roles
			for _, role := range roles {
				role["pivot"] = domain.Record{"admin_id": domain.Integer(user["id"]), "role_id": role["id"]}
			}
			data["menus"] = menus
			return domain.Success(data, "Login berhasil"), nil
		}
		return domain.Success(data, "Login Success"), nil
	case "register":
		return s.registerCustomer(ctx, c, false)
	case "social-register", "social-login":
		return s.social(ctx, c)
	case "me":
		user, e := s.Find(ctx, table, c.Principal.ID)
		if e != nil {
			return domain.Result{}, e
		}
		if kind == "admin" {
			user = secure(user)
			roles, e := s.Roles(ctx, c.Principal.ID)
			if e != nil {
				return domain.Result{}, e
			}
			user["role"] = nil
			if len(roles) > 0 {
				user["role"] = roles[0]["name"]
			}
			return domain.Success(user, "Data user"), nil
		}
		return domain.Success(domain.Pick(user, "id", "name", "full_name", "email", "phone_number", "avatar", "provider", "email_verified_at", "status"), "Data user"), nil
	case "logout":
		_, e := s.Store.Insert(ctx, "point_go_revoked_tokens", domain.Record{"jti": c.Principal.JTI, "expires_at": c.Principal.Expires})
		return domain.Success(nil, "Logout berhasil"), e
	case "change-password", "set-password", "reset-password":
		rules := map[string]string{"old_password": "required", "new_password": "required|string|min:8|max:72", "confirm_password": "required|same:new_password"}
		if c.Action == "set-password" {
			delete(rules, "old_password")
			rules["setup_token"] = "required|string"
			rules["email"] = "required|email"
		}
		if c.Action == "reset-password" {
			rules = map[string]string{"token": "required|string", "email": "required|email", "password": "required|string|min:8|max:72|confirmed"}
		}
		if e := s.validate(ctx, in, rules, ""); e != nil {
			return domain.Result{}, e
		}
		password := domain.Text(in["new_password"])
		if c.Action == "reset-password" {
			password = domain.Text(in["password"])
		}
		hash, e := s.Passwords.Hash(password)
		if e != nil {
			return domain.Result{}, e
		}
		if c.Action == "change-password" {
			user, e := s.Find(ctx, table, c.Principal.ID)
			if e != nil {
				return domain.Result{}, e
			}
			if !s.Passwords.Check(domain.Text(in["old_password"]), domain.Text(user["password"])) {
				return domain.Result{}, domain.Fail("Password lama salah", nil)
			}
			e = s.Store.Update(ctx, table, idQuery(user["id"]), domain.Record{"password": hash})
			return domain.Success(nil, "Password berhasil diubah"), e
		}
		email := domain.Text(in["email"])
		tokenKind, tokenField := "setup", "setup_token"
		if c.Action == "reset-password" {
			tokenKind = "reset"
			tokenField = "token"
		}
		e = s.consumeToken(ctx, tokenKind, email, domain.Text(in[tokenField]), func(db domain.Store) error {
			changes := domain.Record{"password": hash}
			if tokenKind == "setup" {
				changes["verify"] = 1
				changes["email_verified_at"] = time.Now().UTC()
			}
			return db.Update(ctx, table, domain.Q(domain.Equal("email", email)), changes)
		})
		if e != nil {
			return domain.Result{}, e
		}
		if tokenKind == "reset" {
			return domain.Success(nil, "Password berhasil direset."), nil
		}
		user, e := s.Store.Find(ctx, table, domain.Q(domain.Equal("email", email)))
		if e != nil {
			return domain.Result{}, e
		}
		data, e := s.loginData(user, "customer")
		return domain.Result{Body: domain.Record{"success": true, "message": "Password berhasil diatur. Akun Anda telah aktif.", "data": data}, Status: 200}, e
	case "resend-verification", "verify-email", "forgot-password":
		rules := map[string]string{"email": "required|email"}
		if c.Action == "verify-email" {
			rules["token"] = "required|string"
		}
		if e := s.validate(ctx, in, rules, ""); e != nil {
			return domain.Result{}, e
		}
		email := domain.Text(in["email"])
		user, e := s.Optional(ctx, table, domain.Q(domain.Equal("email", email)))
		if e != nil {
			return domain.Result{}, e
		}
		if user == nil {
			return domain.Result{}, domain.Reject(404, domain.Record{"success": false, "message": "Email tidak ditemukan"})
		}
		if c.Action == "forgot-password" {
			token, e := s.putToken(ctx, "reset", email, time.Hour)
			if e != nil {
				return domain.Result{}, e
			}
			link := s.FrontendURL + "/reset-password?token=" + url.QueryEscape(token) + "&email=" + url.QueryEscape(email)
			if e = s.Mail.Send(ctx, email, "Reset Password Notification", link); e != nil {
				return domain.Result{}, e
			}
			return domain.Success(nil, "Reset link berhasil dikirim."), nil
		}
		if domain.Integer(user["verify"]) == 1 {
			return domain.Result{}, domain.Reject(400, domain.Record{"success": false, "message": "Akun Anda sudah terverifikasi. Silakan login."})
		}
		if c.Action == "verify-email" {
			e = s.consumeToken(ctx, "verify", email, domain.Text(in["token"]), func(db domain.Store) error {
				return db.Update(ctx, table, idQuery(user["id"]), domain.Record{"verify": 1, "email_verified_at": time.Now().UTC()})
			})
			if e != nil {
				return domain.Result{}, e
			}
			return domain.Result{Status: 200, Body: domain.Record{"success": true, "message": "Email Anda berhasil diverifikasi. Silakan login.", "email": email}}, nil
		}
		token, e := s.putToken(ctx, "verify", email, 24*time.Hour)
		if e != nil {
			return domain.Result{}, e
		}
		link := s.FrontendURL + "/verify-email?email=" + url.QueryEscape(email) + "&token=" + url.QueryEscape(token)
		if e = s.Mail.Send(ctx, email, "Verifikasi Email", link); e != nil {
			return domain.Result{}, e
		}
		return domain.Result{Status: 200, Body: domain.Record{"success": true, "message": "Link verifikasi telah dikirim ke " + email, "email": email}}, nil
	}
	return domain.Result{}, fmt.Errorf("unknown auth operation")
}

func (s *Service) registerCustomer(ctx context.Context, c domain.Command, admin bool) (domain.Result, error) {
	in := c.Input
	rules := map[string]string{"name": "required|string|max:50", "full_name": "nullable|string|max:50", "email": "required|email|max:50", "phone_number": "required|string|max:15"}
	if admin {
		rules["full_name"] = "required|string|max:50"
		rules["verify"] = "nullable|integer|min:0|max:1"
		rules["address"] = "nullable|string"
		rules["note"] = "nullable|string|max:255"
	}
	if e := s.validate(ctx, in, rules, ""); e != nil {
		return domain.Result{}, e
	}
	for _, field := range []string{"email", "phone_number"} {
		n, e := s.Store.Count(ctx, "customers", domain.Q(domain.Equal(field, in[field])))
		if e != nil {
			return domain.Result{}, e
		}
		if n > 0 {
			if admin {
				return domain.Result{}, domain.Fail(field+" sudah digunakan", in[field])
			}
			return domain.Result{}, domain.Reject(409, domain.Record{"success": false, "message": field + " is already registered", "errors": domain.Record{field: []string{field + " sudah terdaftar, silakan gunakan yang lain."}}, "duplicate_fields": []string{field}, "status": 409})
		}
	}
	temp, e := domain.RandomToken()
	if e != nil {
		return domain.Result{}, e
	}
	temp = temp[:10]
	hash, e := s.Passwords.Hash(temp)
	if e != nil {
		return domain.Result{}, e
	}
	data := domain.Pick(in, "name", "full_name", "email", "phone_number")
	if domain.Text(data["full_name"]) == "" {
		data["full_name"] = data["name"]
	}
	data["password"] = hash
	data["status"] = "active"
	data["verify"] = 0
	data["time"] = time.Now().UTC()
	if admin {
		if domain.Text(in["status"]) != "" {
			data["status"] = in["status"]
		}
		data["verify"] = domain.Integer(in["verify"])
	}
	var user, address domain.Record
	e = s.Store.Transaction(ctx, func(db domain.Store) error {
		var e error
		user, e = db.Insert(ctx, "customers", data)
		if e != nil {
			return e
		}
		if domain.Text(in["address"]) != "" {
			address, e = db.Insert(ctx, "address_text", domain.Record{"customer_id": user["id"], "address": in["address"], "note": in["note"], "status": "active"})
			if e != nil {
				return e
			}
		}
		// The account is committed only after the welcome message is accepted by
		// SMTP. A failed send rolls back the newly created account and address.
		return s.Mail.Send(ctx, domain.Text(user["email"]), "Selamat Datang di Casa Italia Living", fmt.Sprintf("Halo %s,\nEmail: %s\nPassword sementara: %s\nSilakan login dan atur password baru.\n%s", domain.Text(user["full_name"]), domain.Text(user["email"]), temp, s.FrontendURL))
	})
	if e != nil {
		return domain.Result{}, e
	}
	if admin {
		formatted, e := s.resource(ctx, "customers", user)
		if e != nil {
			return domain.Result{}, e
		}
		return domain.Success(domain.Record{"customer": formatted, "generated_password": temp, "address_text": address}, "Customer berhasil ditambahkan"), nil
	}
	return domain.Success(domain.Record{"id": user["id"], "username": user["name"], "email": user["email"]}, "Registrasi berhasil. Password sementara telah dikirim ke email Anda. Silakan cek inbox untuk mengaktifkan akun."), nil
}
func (s *Service) social(ctx context.Context, c domain.Command) (domain.Result, error) {
	if e := s.validate(ctx, c.Input, map[string]string{"provider": "required|string|in:google", "phone_number": "nullable|string|max:15"}, ""); e != nil {
		return domain.Result{}, e
	}
	id, e := s.Google.Verify(ctx, domain.Text(c.Input["id_token"]), domain.Text(c.Input["code"]))
	if e != nil {
		return domain.Result{}, domain.Fail("Unauthorized", "Google token tidak valid")
	}
	user, e := s.Optional(ctx, "customers", domain.Q(domain.Equal("provider", "google"), domain.Equal("provider_id", id.Subject)))
	if e != nil {
		return domain.Result{}, e
	}
	if user == nil {
		user, e = s.Optional(ctx, "customers", domain.Q(domain.Equal("email", id.Email)))
		if e != nil {
			return domain.Result{}, e
		}
	}
	if user == nil {
		if c.Action != "social-register" {
			return domain.Result{}, domain.Fail("Unauthorized", "Akun social belum terdaftar. Silakan register social terlebih dahulu.")
		}
		phone := domain.Text(c.Input["phone_number"])
		if phone != "" {
			n, e := s.Store.Count(ctx, "customers", domain.Q(domain.Equal("phone_number", phone)))
			if e != nil {
				return domain.Result{}, e
			}
			if n > 0 {
				return domain.Result{}, domain.Reject(409, domain.Record{"success": false, "message": "Phone number is already registered", "errors": domain.Record{"phone_number": []string{"Nomor telepon sudah terdaftar, silakan gunakan nomor lain."}}, "duplicate_fields": []string{"phone_number"}, "status": 409})
			}
		}
		if phone == "" {
			token, e := domain.RandomToken()
			if e != nil {
				return domain.Result{}, e
			}
			phone = "9"
			for _, b := range []byte(token[:14]) {
				phone += fmt.Sprint(int(b) % 10)
			}
		}
		temporaryPassword, e := domain.RandomToken()
		if e != nil {
			return domain.Result{}, e
		}
		hash, e := s.Passwords.Hash(temporaryPassword)
		if e != nil {
			return domain.Result{}, e
		}
		user, e = s.Store.Insert(ctx, "customers", domain.Record{"name": id.Name, "full_name": id.Name, "email": id.Email, "phone_number": phone, "password": hash, "status": "active", "verify": 1, "email_verified_at": time.Now().UTC(), "provider": "google", "provider_id": id.Subject, "avatar": id.Avatar, "time": time.Now().UTC()})
		if e != nil {
			return domain.Result{}, e
		}
	} else {
		e = s.Store.Update(ctx, "customers", idQuery(user["id"]), domain.Record{"name": id.Name, "provider": "google", "provider_id": id.Subject, "avatar": id.Avatar, "email_verified_at": time.Now().UTC(), "time": time.Now().UTC()})
		if e != nil {
			return domain.Result{}, e
		}
	}
	data, e := s.loginData(user, "customer")
	data["username"] = id.Name
	data["provider"] = "google"
	message := "Social login customer berhasil"
	if c.Action == "social-register" {
		message = "Social auth customer berhasil"
	}
	return domain.Success(data, message), e
}
