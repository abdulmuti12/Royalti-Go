package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/abdulmuti12/api-point-go/internal/domain"
)

func (s *Service) resource(ctx context.Context, t string, r domain.Record) (domain.Record, error) {
	fields := map[string][]string{
		"admins":        {"id", "name", "email", "status", "last_login", "created_at", "updated_at"},
		"roles":         {"id", "name", "created_at", "updated_at"},
		"customers":     {"id", "name", "full_name", "email", "phone_number", "status", "verify"},
		"category":      {"id", "name", "created_at", "image"},
		"brands":        {"id", "name", "note", "image", "description", "created_at", "updated_at", "link"},
		"banner":        {"id", "name", "decription", "note", "type", "file", "file2", "file3", "file4", "created_at", "updated_at"},
		"promotions":    {"id", "name", "description", "note", "type", "status", "file", "file2", "file3", "file4", "file5", "created_at", "updated_at"},
		"logos":         {"id", "description", "image", "is_active"},
		"points":        {"id", "name", "status", "price_point", "range_point", "point", "description", "created_at", "updated_at"},
		"gifts":         {"id", "name", "total_point", "image", "description", "status", "created_at", "updated_at"},
		"member_levels": {"id", "name", "min_point", "max_point", "file", "description", "status", "created_at", "updated_at"},
		"contact_us":    {"id", "name", "email", "phone_number", "description", "created_at", "updated_at"},
		"transaction":   {"id", "transaction_code", "order_id", "total_payment", "status", "note", "address_id", "created_at", "updated_at"},
		"orders":        {"id", "order_code", "id_product", "qty", "status", "customer_id", "description", "created_at", "updated_at"},
	}
	o := domain.Pick(r, fields[t]...)
	switch t {
	case "admins":
		roles, e := s.Roles(ctx, domain.Text(r["id"]))
		if e != nil {
			return nil, e
		}
		o["role"] = nil
		if len(roles) > 0 {
			o["role"] = roles[0]["name"]
		}
	case "customers":
		o["verify"] = domain.Integer(r["verify"])
		rows, e := s.Store.List(ctx, "address_text", customerQuery(r["id"]))
		if e != nil {
			return nil, e
		}
		o["address_text"] = []domain.Record{}
		for _, a := range rows {
			o["address_text"] = append(o["address_text"].([]domain.Record), domain.Pick(a, "id", "address", "note"))
		}
	case "category", "logos":
		o["image"] = s.Asset(r["image"])
		if t == "logos" {
			o["is_active"] = domain.Bool(r["is_active"])
		}
	case "promotions":
		brand, e := s.Optional(ctx, "brands", idQuery(r["brand_id"]))
		if e != nil {
			return nil, e
		}
		o["brand"] = nil
		if brand != nil {
			o["brand"] = brand["name"]
		}
		o["file"] = s.Asset(r["file"])
	case "points":
		for _, field := range []string{"price_point", "range_point", "point"} {
			o[field] = domain.Integer(r[field])
		}
	}
	return o, nil
}
func (s *Service) detail(ctx context.Context, t string, r domain.Record) (any, error) {
	switch t {
	case "admins":
		o, e := s.resource(ctx, t, r)
		return domain.Record{"general": o}, e
	case "customers":
		return domain.Record{"general": domain.Pick(r, "id", "name", "full_name", "email", "phone_number", "status", "time", "created_at", "updated_at")}, nil
	case "brands":
		o, _ := s.resource(ctx, t, r)
		o["image"] = s.Asset(r["image"])
		return domain.Record{"general": o}, nil
	case "promotions":
		o, e := s.resource(ctx, t, r)
		if e != nil {
			return nil, e
		}
		delete(o, "status")
		for _, f := range []string{"file2", "file3", "file4", "file5"} {
			if domain.Text(r["file"]) != "" {
				o[f] = s.Asset(r[f])
				if o[f] == nil {
					o[f] = strings.TrimRight(s.AppURL, "/") + "/storage"
				}
			} else {
				o[f] = nil
			}
		}
		return domain.Record{"general": o}, nil
	case "banner":
		o := domain.Pick(r, "id", "name", "description", "note", "type", "title", "title2", "title3", "title4", "title5", "created_at", "updated_at")
		for _, f := range []string{"file", "file2", "file3", "file4", "file5", "file6", "file7"} {
			o[f] = s.Asset(r[f])
		}
		return domain.Record{"general": o}, nil
	case "contact_us":
		o, e := s.resource(ctx, t, r)
		return domain.Record{"general": o}, e
	case "roles":
		o, _ := s.resource(ctx, t, r)
		pivots, e := s.Store.List(ctx, "role_menu", domain.Q(domain.Equal("role_id", r["id"])))
		if e != nil {
			return nil, e
		}
		rows, e := s.Store.List(ctx, "menus", domain.Q(domain.In("id", listIDs(pivots, "menu_id"))))
		if e != nil {
			return nil, e
		}
		menus := []domain.Record{}
		for _, m := range rows {
			menus = append(menus, domain.Pick(m, "id", "name", "route"))
		}
		o["menu"] = domain.Record{"menu": menus}
		return domain.Record{"general": o}, nil
	case "category", "logos":
		return secure(r), nil
	default:
		return s.resource(ctx, t, r)
	}
}

func (s *Service) catalog(ctx context.Context, c domain.Command) (domain.Result, error) {
	t, in := c.Table, c.Input
	spec, ok := catalogs[t]
	if !ok {
		return domain.Result{}, fmt.Errorf("unknown catalog")
	}
	if c.Action == "create" && t == "customers" {
		return s.registerCustomer(ctx, c, true)
	}
	if c.Action == "index" {
		q := domain.Query{Sort: []domain.Sort{{Field: "id", Desc: true}}}
		for _, f := range []string{"name", "full_name", "email", "phone_number", "description", "transaction_code"} {
			if domain.Text(in[f]) != "" {
				q.Filters = append(q.Filters, domain.Filter{Field: f, Op: "LIKE", Value: "%" + domain.Text(in[f]) + "%"})
			}
		}
		if t == "customers" && domain.Text(in["search"]) != "" {
			q.AnyLike = []string{"name", "full_name", "email", "phone_number"}
			q.Keyword = domain.Text(in["search"])
		}
		if (t == "gifts" || t == "member_levels") && domain.Text(in["status"]) != "" {
			q.Filters = append(q.Filters, domain.Equal("status", in["status"]))
		}
		if t == "orders" {
			rows, e := s.Store.List(ctx, "transaction", domain.Q(domain.Equal("status", "success")))
			if e != nil {
				return domain.Result{}, e
			}
			q.Filters = append(q.Filters, domain.In("id", listIDs(rows, "order_id")))
			if domain.Text(in["order_code"]) != "" {
				q.Filters = append(q.Filters, domain.Filter{Field: "order_number", Op: "LIKE", Value: "%" + domain.Text(in["order_code"]) + "%"})
			}
		}
		page, e := s.page(ctx, t, q, in, spec.Style, 10)
		if e != nil {
			return domain.Result{}, e
		}
		for i, r := range page.Items {
			page.Items[i], e = s.resource(ctx, t, r)
			if e != nil {
				return domain.Result{}, e
			}
		}
		msg := "Data berhasil diambil"
		if t == "admins" {
			msg = "Success Load Data"
		}
		return domain.Success(page, msg), nil
	}
	var row domain.Record
	if c.Action != "create" {
		var e error
		row, e = s.Find(ctx, t, c.ID)
		if e != nil {
			return domain.Result{}, missing(e, "Data tidak ditemukan", "Data tidak ditemukan")
		}
	}
	switch c.Action {
	case "show":
		data, e := s.detail(ctx, t, row)
		msg := "Success Load Data"
		if t == "banner" {
			msg = "Load Banner Success"
		}
		return domain.Success(data, msg), e
	case "edit":
		var data any
		var e error
		msg := "Data berhasil diambil"
		switch t {
		case "admins":
			data, e = s.detail(ctx, t, row)
			msg = "Success Load Data"
		case "banner":
			data = row
			msg = "Load Banner Success"
		case "promotions":
			data = domain.Copy(row)
			for _, f := range []string{"file", "file2", "file3", "file4", "file5"} {
				data.(domain.Record)[f] = s.Asset(row[f])
			}
			msg = "Success Load Data"
		default:
			data, e = s.resource(ctx, t, row)
		}
		return domain.Success(data, msg), e
	case "role-edit":
		menus, e := s.Store.List(ctx, "menus", domain.Query{})
		if e != nil {
			return domain.Result{}, e
		}
		pivots, e := s.Store.List(ctx, "role_menu", domain.Q(domain.Equal("role_id", row["id"])))
		if e != nil {
			return domain.Result{}, e
		}
		selected := map[string]bool{}
		for _, p := range pivots {
			selected[domain.Text(p["menu_id"])] = true
		}
		data := []domain.Record{}
		for _, m := range menus {
			o := domain.Pick(m, "id", "name")
			o["selected"] = selected[domain.Text(m["id"])]
			data = append(data, o)
		}
		return domain.Success(domain.Record{"name": row["name"], "menus": data}, "Data LoadSuccess"), nil
	case "create", "update", "set-active":
		rules := spec.Create
		if c.Action == "update" {
			rules = spec.Update
		}
		if len(rules) == 0 {
			rules = spec.Create
		}
		if c.Action == "set-active" {
			rules = nil
		}
		if t == "contact_us" {
			rules = map[string]string{"name": "required|string|max:255", "email": "required|email|max:255", "phone_number": "nullable|string|max:20", "description": "nullable|string|max:1000"}
		}
		if t == "transaction" {
			rules = map[string]string{"order_id": "required|integer|min:1|exists:orders,id", "total_payment": "required|integer|min:0", "status": "required|in:pending,success,failed,cancelled", "note": "nullable|string", "address_id": "required|integer|min:1|exists:addresses,id"}
		}
		if t == "customers" {
			rules = map[string]string{"name": "sometimes|string|max:50", "full_name": "sometimes|string|max:50", "email": "sometimes|email|max:50|unique:customers,email", "phone_number": "sometimes|string|max:15|unique:customers,phone_number", "status": "nullable|string|max:20", "verify": "nullable|integer|min:0|max:1"}
		}
		if e := s.validate(ctx, in, withoutFiles(rules), c.ID); e != nil {
			return domain.Result{}, e
		}
		if t == "points" && (domain.Integer(in["range_point"]) <= 0 || domain.Integer(in["point"]) < 0) {
			return domain.Result{}, domain.Fail("Validasi gagal", domain.Record{"range_point": []string{"range_point harus lebih dari 0 dan point tidak boleh negatif."}})
		}
		if t == "gifts" && domain.Integer(in["total_point"]) < 0 {
			return domain.Result{}, domain.Fail("Validasi gagal", "total_point tidak boleh negatif")
		}
		if t == "admins" {
			if e := s.validate(ctx, in, map[string]string{"role_id": "required|exists:roles,id"}, ""); e != nil {
				return domain.Result{}, e
			}
		}
		if t == "roles" {
			if e := s.validateMenus(ctx, in["menus"]); e != nil {
				return domain.Result{}, e
			}
		}
		if t != "admins" && t != "customers" && t != "promotions" && t != "banner" && t != "logos" && t != "transaction" && t != "orders" && domain.Text(in["name"]) != "" {
			q := domain.Q(domain.Equal("name", in["name"]))
			if c.ID != "" {
				q.Filters = append(q.Filters, domain.NotEqual("id", c.ID))
			}
			n, e := s.Store.Count(ctx, t, q)
			if e != nil {
				return domain.Result{}, e
			}
			if n > 0 {
				return domain.Result{}, domain.Fail("Gagal", spec.Title+" sudah ada")
			}
		}
		changes := domain.Record{}
		for _, f := range spec.Fields {
			if v, exists := in[f]; exists {
				changes[f] = v
			} else if c.Action == "create" {
				changes[f] = nil
			}
		}
		delete(changes, "last_login")
		if t == "transaction" && c.Action == "create" {
			token, e := domain.RandomToken()
			if e != nil {
				return domain.Result{}, e
			}
			changes["transaction_code"] = "TRX-" + token[:16]
		}
		if t == "transaction" && c.Action == "update" {
			delete(changes, "transaction_code")
		}
		if t == "customers" {
			delete(changes, "provider")
			delete(changes, "provider_id")
			delete(changes, "email_verified_at")
		}
		if t == "admins" {
			hash, e := s.Passwords.Hash(domain.Text(in["password"]))
			if e != nil {
				return domain.Result{}, e
			}
			changes["password"] = hash
		}
		if t == "points" || t == "gifts" || t == "member_levels" {
			if domain.Text(changes["status"]) == "" {
				changes["status"] = "active"
			}
		}
		if t == "logos" {
			if c.Action == "set-active" {
				changes = domain.Record{"is_active": !domain.Bool(row["is_active"])}
			} else {
				changes["is_active"] = true
			}
		}
		e := s.Store.Transaction(ctx, func(db domain.Store) error {
			if t == "points" || t == "promotions" || t == "banner" || t == "logos" {
				if _, e := db.Find(ctx, "point_go_locks", domain.Query{Filters: []domain.Filter{domain.Equal("key", t)}, Lock: true}); e != nil {
					return e
				}
			}
			var e error
			if c.Action == "create" {
				row, e = db.Insert(ctx, t, changes)
			} else {
				e = db.Update(ctx, t, idQuery(c.ID), changes)
				if e == nil {
					row, e = db.Find(ctx, t, idQuery(c.ID))
				}
			}
			if e != nil {
				return e
			}
			if t == "admins" {
				if e := db.Delete(ctx, "admin_role", domain.Q(domain.Equal("admin_id", row["id"]))); e != nil {
					return e
				}
				if _, e := db.Insert(ctx, "admin_role", domain.Record{"admin_id": row["id"], "role_id": in["role_id"]}); e != nil {
					return e
				}
			}
			if t == "roles" {
				if e := db.Delete(ctx, "role_menu", domain.Q(domain.Equal("role_id", row["id"]))); e != nil {
					return e
				}
				for _, id := range slice(in["menus"]) {
					if _, e := db.Insert(ctx, "role_menu", domain.Record{"role_id": row["id"], "menu_id": id}); e != nil {
						return e
					}
				}
			}
			if (t == "points" && strings.EqualFold(domain.Text(row["status"]), "active")) || (t == "promotions" && domain.Text(row["status"]) == "active") || (t == "banner" && domain.Text(row["status"]) == "Active") || (t == "logos" && domain.Bool(row["is_active"])) {
				other := domain.Record{"status": "inactive"}
				if t == "points" {
					other["status"] = "non-active"
				}
				if t == "banner" {
					other["status"] = "Non Active"
				}
				if t == "logos" {
					other = domain.Record{"is_active": false}
				}
				return db.Update(ctx, t, domain.Q(domain.NotEqual("id", row["id"])), other)
			}
			return nil
		})
		if e != nil {
			return domain.Result{}, e
		}
		data := any(secure(row))
		if t == "logos" {
			data, e = s.resource(ctx, t, row)
			if e != nil {
				return domain.Result{}, e
			}
		}
		messages := map[string][2]string{"admins": {"Register berhasil", "Update berhasil"}, "roles": {"Tambah Roles berhasil", "Role updated successfully."}, "category": {"Tambah Kategori berhasil", "Update Kategori berhasil"}, "brands": {"Data berhasil ditambahkan", "Update Kategori berhasil"}, "banner": {"Add Banner Success", "Banner updated successfully"}, "promotions": {"Add PromotionSuccess", "Promotion updated successfully"}, "logos": {"Data berhasil ditambahkan", "Data berhasil diperbarui"}}
		msg := "Tambah " + spec.Title + " berhasil"
		if c.Action == "update" {
			msg = "Update " + spec.Title + " berhasil"
		}
		if pair, ok := messages[t]; ok {
			msg = pair[0]
			if c.Action == "update" {
				msg = pair[1]
			}
		}
		if c.Action == "set-active" {
			msg = "Logo berhasil diaktifkan"
			if !domain.Bool(row["is_active"]) {
				msg = "Logo berhasil dinonaktifkan"
			}
		}
		if t == "roles" && c.Action == "update" {
			return domain.Result{Status: 200, Body: domain.Record{"success": true, "message": msg, "data": data}}, nil
		}
		return domain.Success(data, msg), nil
	case "delete":
		e := s.Store.Transaction(ctx, func(db domain.Store) error {
			if t == "customers" {
				if e := s.lockCustomer(ctx, db, c.ID); e != nil {
					return e
				}
				n, e := db.Count(ctx, "orders", customerQuery(c.ID))
				if e != nil {
					return e
				}
				if n > 0 {
					return domain.Fail("Customer memiliki order", "Customer tidak dapat dihapus selama masih memiliki order")
				}
				for _, related := range []string{"gift_claims", "redeem_histories", "redeem_points", "address_text"} {
					if e := db.Delete(ctx, related, customerQuery(c.ID)); e != nil {
						return e
					}
				}
				if e := db.Delete(ctx, "addresses", domain.Q(domain.Equal("user_id", c.ID))); e != nil {
					return e
				}
			}
			if t == "gifts" {
				n, e := db.Count(ctx, "gift_claims", domain.Q(domain.Equal("gift_id", c.ID)))
				if e != nil {
					return e
				}
				if n > 0 {
					return domain.Fail("Gift masih digunakan", "Gift dengan riwayat claim tidak dapat dihapus")
				}
			}
			if t == "admins" {
				if e := db.Delete(ctx, "admin_role", domain.Q(domain.Equal("admin_id", c.ID))); e != nil {
					return e
				}
			}
			if t == "roles" {
				if e := db.Delete(ctx, "role_menu", domain.Q(domain.Equal("role_id", c.ID))); e != nil {
					return e
				}
				if e := db.Delete(ctx, "admin_role", domain.Q(domain.Equal("role_id", c.ID))); e != nil {
					return e
				}
			}
			return db.Delete(ctx, t, idQuery(c.ID))
		})
		if e != nil {
			return domain.Result{}, e
		}
		msg := "Hapus " + spec.Title + " berhasil"
		if t == "admins" {
			msg = "Delete berhasil"
		}
		if t == "promotions" || t == "contact_us" {
			msg = "Data berhasil dihapus"
		}
		if t == "banner" {
			msg = "Data dan file berhasil dihapus"
		}
		return domain.Success(nil, msg), nil
	}
	return domain.Result{}, errors.New("unknown catalog operation")
}
func slice(v any) []any {
	if a, ok := v.([]any); ok {
		return a
	}
	if v == nil {
		return nil
	}
	return []any{v}
}
func (s *Service) validateMenus(ctx context.Context, v any) error {
	seen := map[string]bool{}
	for _, id := range slice(v) {
		if seen[domain.Text(id)] {
			return domain.Fail("Validasi gagal", "Menu duplikat")
		}
		seen[domain.Text(id)] = true
		n, e := s.Store.Count(ctx, "menus", idQuery(id))
		if e != nil {
			return e
		}
		if n == 0 {
			return domain.Fail("Validasi gagal", "Menu tidak ditemukan")
		}
	}
	return nil
}
