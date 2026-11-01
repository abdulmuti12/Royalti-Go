package application

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/abdulmuti12/api-point-go/internal/domain"
)

func (s *Service) redeemView(ctx context.Context, r domain.Record) (domain.Record, error) {
	c, e := s.Optional(ctx, "customers", idQuery(r["customer_id"]))
	if e != nil {
		return nil, e
	}
	p, e := s.Optional(ctx, "points", idQuery(r["point_id"]))
	if e != nil {
		return nil, e
	}
	out := domain.Pick(r, "id", "total_transaction", "total_point_active", "total_point_closed", "created_at", "updated_at")
	out["total_point"] = domain.Integer(r["total_point_active"]) + domain.Integer(r["total_point_closed"])
	out["customer"] = nil
	out["point"] = nil
	if c != nil {
		out["customer"] = domain.Pick(c, "id", "name", "full_name", "email", "phone_number")
	}
	if p != nil {
		out["point"] = domain.Pick(p, "id", "name", "status", "price_point", "range_point", "point")
	}
	return out, nil
}
func (s *Service) historyView(ctx context.Context, r domain.Record) (domain.Record, error) {
	out := domain.Pick(r, "id", "customer_id", "point_id", "type", "point_amount", "point_balance", "status", "description", "transaction_amount")
	out["date"] = dateISO(r["created_at"])
	out["point"] = nil
	p, e := s.Optional(ctx, "points", idQuery(r["point_id"]))
	if e != nil {
		return nil, e
	}
	if p != nil {
		out["point"] = domain.Pick(p, "id", "name", "range_point", "point")
	}
	return out, nil
}
func (s *Service) claimView(ctx context.Context, r domain.Record) (domain.Record, error) {
	out := domain.Pick(r, "id", "customer_id", "gift_id", "redeem_point_id", "approved_by", "required_point", "customer_point_at_request", "status", "notes", "admin_note", "approved_at", "completed_at", "created_at", "updated_at")
	for _, field := range []string{"approved_at", "completed_at", "created_at", "updated_at"} {
		out[field] = dateISO(out[field])
	}
	for _, rel := range []struct {
		Field, Table, FK string
		Fields           []string
	}{{"customer", "customers", "customer_id", []string{"id", "name", "full_name", "email", "phone_number"}}, {"gift", "gifts", "gift_id", []string{"id", "name", "total_point", "image", "image2", "image3", "description"}}, {"approver", "admins", "approved_by", []string{"id", "name", "email"}}} {
		row, e := s.Optional(ctx, rel.Table, idQuery(r[rel.FK]))
		if e != nil {
			return nil, e
		}
		out[rel.Field] = nil
		if row != nil {
			out[rel.Field] = domain.Pick(row, rel.Fields...)
		}
	}
	return out, nil
}

func (s *Service) loyalty(ctx context.Context, c domain.Command) (domain.Result, error) {
	switch c.Action {
	case "redeems", "point-history", "gift-claims":
		table, style, msg := "redeem_points", "nested", "Data berhasil diambil"
		if c.Action == "point-history" {
			table = "redeem_histories"
			msg = "History berhasil diambil"
		}
		if c.Action == "gift-claims" {
			table = "gift_claims"
			msg = "Data gift claims berhasil diambil"
		}
		q := domain.Query{Sort: []domain.Sort{{Field: "id", Desc: true}}}
		if c.Action == "point-history" {
			if _, e := s.Find(ctx, "customers", c.ID); e != nil {
				return domain.Result{}, missing(e, "Customer tidak ditemukan", "Data tidak ditemukan")
			}
			q.Filters = append(q.Filters, domain.Equal("customer_id", c.ID))
		}
		for _, f := range []string{"status", "type", "gift_id"} {
			if domain.Text(c.Input[f]) != "" {
				q.Filters = append(q.Filters, domain.Equal(f, c.Input[f]))
			}
		}
		q, e := s.relatedCustomer(ctx, q, domain.Text(c.Input["customer_name"]))
		if e != nil {
			return domain.Result{}, e
		}
		page, e := s.page(ctx, table, q, c.Input, style, 10)
		if e != nil {
			return domain.Result{}, e
		}
		for i, r := range page.Items {
			switch c.Action {
			case "redeems":
				page.Items[i], e = s.redeemView(ctx, r)
			case "point-history":
				page.Items[i], e = s.historyView(ctx, r)
			case "gift-claims":
				page.Items[i], e = s.claimView(ctx, r)
			}
			if e != nil {
				return domain.Result{}, e
			}
		}
		return domain.Success(page, msg), nil
	case "redeem-detail":
		user, e := s.Find(ctx, "customers", c.ID)
		if e != nil {
			return domain.Result{}, missing(e, "Customer tidak ditemukan", "Data tidak ditemukan")
		}
		q := customerQuery(c.ID)
		q.Sort = []domain.Sort{{Field: "id", Desc: true}}
		rows, e := s.Store.List(ctx, "redeem_points", q)
		if e != nil {
			return domain.Result{}, e
		}
		summary := domain.Record{"customer": domain.Pick(user, "id", "name", "full_name", "email", "phone_number"), "total_point": sum(rows, "total_point_active") + sum(rows, "total_point_closed"), "total_point_active": sum(rows, "total_point_active"), "total_point_closed": sum(rows, "total_point_closed"), "total_earned": sum(rows, "total_point_earned"), "total_claimed": sum(rows, "total_point_claimed")}
		redeems := []domain.Record{}
		for _, r := range rows {
			view, e := s.redeemView(ctx, r)
			if e != nil {
				return domain.Result{}, e
			}
			redeems = append(redeems, view)
		}
		hist, e := s.Store.List(ctx, "redeem_histories", q)
		if e != nil {
			return domain.Result{}, e
		}
		all, active, closed := []domain.Record{}, []domain.Record{}, []domain.Record{}
		for _, r := range hist {
			view, e := s.historyView(ctx, r)
			if e != nil {
				return domain.Result{}, e
			}
			all = append(all, view)
			if domain.Text(r["status"]) == "active" {
				active = append(active, view)
			} else {
				closed = append(closed, view)
			}
		}
		return domain.Success(domain.Record{"summary": summary, "redeems": redeems, "history": domain.Record{"all": all, "active": active, "closed": closed}}, "Detail customer berhasil diambil"), nil
	case "calculate":
		return s.calculate(ctx, c)
	case "claim-points":
		return s.claimPoints(ctx, c)
	case "member-level":
		return s.memberLevel(ctx, c.Principal.ID)
	case "gifts":
		return s.gifts(ctx, c)
	case "gift-history":
		q := customerQuery(c.Principal.ID)
		q.Sort = []domain.Sort{{Field: "id", Desc: true}}
		if domain.Text(c.Input["status"]) != "" {
			q.Filters = append(q.Filters, domain.Equal("status", c.Input["status"]))
		}
		rows, e := s.Store.List(ctx, "gift_claims", q)
		if e != nil {
			return domain.Result{}, e
		}
		out := []domain.Record{}
		for _, r := range rows {
			o := domain.Pick(r, "id", "gift_id", "required_point", "status", "notes", "admin_note", "created_at", "approved_at", "completed_at")
			for _, field := range []string{"created_at", "approved_at", "completed_at"} {
				o[field] = dateISO(o[field])
			}
			gift, e := s.Optional(ctx, "gifts", idQuery(r["gift_id"]))
			if e != nil {
				return domain.Result{}, e
			}
			o["gift_name"] = nil
			o["gift_image"] = nil
			o["gift_image2"] = nil
			o["gift_image3"] = nil
			if gift != nil {
				o["gift_name"] = gift["name"]
				o["gift_image"] = s.Asset(gift["image"])
				o["gift_image2"] = s.Asset(gift["image2"])
				o["gift_image3"] = s.Asset(gift["image3"])
			}
			out = append(out, o)
		}
		return domain.Success(domain.Record{"claims": out}, "Data claim berhasil diambil"), nil
	case "claim-gift":
		return s.claimGift(ctx, c)
	case "gift-claim":
		row, e := s.Find(ctx, "gift_claims", c.ID)
		if e != nil {
			return domain.Result{}, missing(e, "Claim tidak ditemukan", "Data tidak ditemukan")
		}
		view, e := s.claimView(ctx, row)
		return domain.Success(view, "Detail claim berhasil diambil"), e
	case "approve-gift", "reject-gift", "complete-gift":
		return s.reviewGift(ctx, c)
	}
	return domain.Result{}, fmt.Errorf("unknown loyalty operation")
}

func (s *Service) calculate(ctx context.Context, c domain.Command) (domain.Result, error) {
	var result domain.Record
	var newPoints int64
	var user domain.Record
	e := s.Store.Transaction(ctx, func(db domain.Store) error {
		if _, e := db.Find(ctx, "point_go_locks", domain.Query{Filters: []domain.Filter{domain.Equal("key", "points")}, Lock: true}); e != nil {
			return e
		}
		if e := s.lockCustomer(ctx, db, c.ID); e != nil {
			return missing(e, "Customer tidak ditemukan", "Data tidak ditemukan")
		}
		var e error
		user, e = db.Find(ctx, "customers", idQuery(c.ID))
		if e != nil {
			return e
		}
		rule, e := db.Find(ctx, "points", domain.Q(domain.Equal("status", "active")))
		if e != nil {
			return missing(e, "Rule point aktif tidak ditemukan", "Tidak ada point rule dengan status active")
		}
		if domain.Integer(rule["range_point"]) <= 0 {
			return domain.Fail("Rule tidak valid", "range_point harus lebih dari 0")
		}
		total, e := s.customerTotal(ctx, db, c.ID)
		if e != nil {
			return e
		}
		if total <= 0 {
			return domain.Fail("Tidak ada transaksi", "Customer belum memiliki transaksi sukses")
		}
		calculated := earned(total, rule)
		if calculated <= 0 {
			return domain.Fail("Belum cukup transaksi", fmt.Sprintf("Total transaksi %d belum memenuhi range_point %d untuk mendapat point", total, domain.Integer(rule["range_point"])))
		}
		q := domain.Q(domain.Equal("customer_id", c.ID), domain.Equal("point_id", rule["id"]))
		q.Lock = true
		rp, e := db.Find(ctx, "redeem_points", q)
		if e != nil && !errorsNotFound(e) {
			return e
		}
		previous := domain.Integer(rp["total_point_earned"])
		newPoints = calculated - previous
		if newPoints <= 0 {
			result = rp
			return nil
		}
		if rp == nil {
			rp, e = db.Insert(ctx, "redeem_points", domain.Record{"customer_id": c.ID, "point_id": rule["id"], "total_transaction": total, "total_point_earned": calculated, "total_point_claimed": 0, "total_point_active": newPoints, "total_point_closed": 0})
			if e != nil {
				return e
			}
		} else {
			changes := domain.Record{"total_transaction": total, "total_point_earned": calculated, "total_point_active": domain.Integer(rp["total_point_active"]) + newPoints}
			if e = db.Update(ctx, "redeem_points", idQuery(rp["id"]), changes); e != nil {
				return e
			}
			for k, v := range changes {
				rp[k] = v
			}
		}
		if _, e = db.Insert(ctx, "redeem_histories", domain.Record{"redeem_point_id": rp["id"], "customer_id": c.ID, "point_id": rule["id"], "type": "earn", "point_amount": newPoints, "point_balance": domain.Integer(rp["total_point_active"]) + domain.Integer(rp["total_point_closed"]), "transaction_amount": total, "status": "active", "description": fmt.Sprintf("Earn %d point dari total transaksi %d (rule: %s)", newPoints, total, domain.Text(rule["name"]))}); e != nil {
			return e
		}
		result = rp
		return s.audit(ctx, db, c.Principal, "point.calculated", "redeem_point", rp["id"], fmt.Sprintf("Earn %d point", newPoints))
	})
	if e != nil {
		return domain.Result{}, e
	}
	if newPoints <= 0 {
		return domain.Success(result, "Tidak ada point baru (point sudah pernah dihitung sebelumnya)"), nil
	}
	view, e := s.redeemView(ctx, result)
	return domain.Success(view, fmt.Sprintf("Berhasil menambahkan %d point untuk customer %s", newPoints, domain.Text(user["name"]))), e
}
func errorsNotFound(e error) bool { return e == domain.ErrNotFound }

func deduct(ctx context.Context, db domain.Store, rp domain.Record, amount int64, description string, activeBalance bool) (domain.Record, error) {
	if amount <= 0 || domain.Integer(rp["total_point_active"]) < amount {
		return nil, domain.Fail("Point aktif tidak cukup", fmt.Sprintf("Point aktif %d, tidak bisa claim %d", domain.Integer(rp["total_point_active"]), amount))
	}
	changes := domain.Record{"total_point_active": domain.Integer(rp["total_point_active"]) - amount, "total_point_closed": domain.Integer(rp["total_point_closed"]) + amount, "total_point_claimed": domain.Integer(rp["total_point_claimed"]) + amount}
	if e := db.Update(ctx, "redeem_points", idQuery(rp["id"]), changes); e != nil {
		return nil, e
	}
	for k, v := range changes {
		rp[k] = v
	}
	balance := domain.Integer(rp["total_point_active"]) + domain.Integer(rp["total_point_closed"])
	if activeBalance {
		balance = domain.Integer(rp["total_point_active"])
	}
	return db.Insert(ctx, "redeem_histories", domain.Record{"redeem_point_id": rp["id"], "customer_id": rp["customer_id"], "point_id": rp["point_id"], "type": "claim", "point_amount": amount, "point_balance": balance, "status": "closed", "description": description})
}
func latestRedeem(id string) domain.Query {
	return domain.Query{Filters: []domain.Filter{domain.Equal("customer_id", id)}, Sort: []domain.Sort{{Field: "created_at", Desc: true}, {Field: "id", Desc: true}}, Lock: true}
}
func (s *Service) claimPoints(ctx context.Context, c domain.Command) (domain.Result, error) {
	if e := s.validate(ctx, c.Input, map[string]string{"customer_id": "required|string|exists:customers,id", "point_amount": "required|integer|min:1", "description": "nullable|string"}, ""); e != nil {
		return domain.Result{}, e
	}
	id := domain.Text(c.Input["customer_id"])
	amount := domain.Integer(c.Input["point_amount"])
	var rp, hist domain.Record
	e := s.Store.Transaction(ctx, func(db domain.Store) error {
		if e := s.lockCustomer(ctx, db, id); e != nil {
			return e
		}
		var e error
		rp, e = db.Find(ctx, "redeem_points", latestRedeem(id))
		if e != nil {
			return missing(e, "Point tidak ditemukan", "Customer belum memiliki point")
		}
		description := domain.Text(c.Input["description"])
		if description == "" {
			user, e := db.Find(ctx, "customers", idQuery(id))
			if e != nil {
				return e
			}
			description = fmt.Sprintf("Claim %d point oleh customer %s", amount, domain.Text(user["name"]))
		}
		hist, e = deduct(ctx, db, rp, amount, description, false)
		if e != nil {
			return e
		}
		return s.audit(ctx, db, c.Principal, "point.claimed", "redeem_point", rp["id"], description)
	})
	if e != nil {
		return domain.Result{}, e
	}
	redeem, e := s.redeemView(ctx, rp)
	if e != nil {
		return domain.Result{}, e
	}
	history, e := s.historyView(ctx, hist)
	return domain.Success(domain.Record{"redeem": redeem, "history": history}, fmt.Sprintf("Claim %d point berhasil", amount)), e
}

func (s *Service) memberLevel(ctx context.Context, id string) (domain.Result, error) {
	rows, e := s.Store.List(ctx, "redeem_points", customerQuery(id))
	if e != nil {
		return domain.Result{}, e
	}
	total := sum(rows, "total_point_earned")
	levels, e := s.Store.List(ctx, "member_levels", domain.Query{Filters: []domain.Filter{domain.Equal("status", "active")}, Sort: []domain.Sort{{Field: "min_point"}}})
	if e != nil {
		return domain.Result{}, e
	}
	var current, next domain.Record
	for _, l := range levels {
		min, max := domain.Integer(l["min_point"]), domain.Integer(l["max_point"])
		if total >= min && (max == 0 || total <= max) {
			current = l
			break
		}
	}
	for _, l := range levels {
		if current != nil && domain.Integer(l["min_point"]) > domain.Integer(current["min_point"]) || current == nil && domain.Integer(l["min_point"]) > total {
			next = l
			break
		}
	}
	format := func(l domain.Record) any {
		if l == nil {
			return nil
		}
		o := domain.Pick(l, "id", "name", "min_point", "max_point", "description")
		o["min_point"] = domain.Integer(l["min_point"])
		o["max_point"] = domain.Integer(l["max_point"])
		o["image_url"] = s.Asset(l["file"])
		return o
	}
	var progress any
	if next != nil {
		base := domain.Integer(current["min_point"])
		target := domain.Integer(next["min_point"])
		rng := target - base
		if rng < 1 {
			rng = 1
		}
		pct := math.Round(float64(total-base)/float64(rng)*10000) / 100
		if pct < 0 {
			pct = 0
		}
		if pct > 100 {
			pct = 100
		}
		progress = domain.Record{"current": total, "min": base, "target": target, "percent": pct}
	}
	all := []any{}
	for _, l := range levels {
		all = append(all, format(l))
	}
	return domain.Success(domain.Record{"total_points": total, "current_level": format(current), "next_level": format(next), "progress": progress, "all_levels": all}, "Data member level berhasil diambil"), nil
}

func (s *Service) gifts(ctx context.Context, c domain.Command) (domain.Result, error) {
	rows, e := s.Store.List(ctx, "redeem_points", customerQuery(c.Principal.ID))
	if e != nil {
		return domain.Result{}, e
	}
	total := sum(rows, "total_point_active")
	gifts, e := s.Store.List(ctx, "gifts", domain.Query{Filters: []domain.Filter{domain.Equal("status", "active")}, Sort: []domain.Sort{{Field: "total_point"}}})
	if e != nil {
		return domain.Result{}, e
	}
	claims, e := s.Store.List(ctx, "gift_claims", domain.Query{Filters: []domain.Filter{domain.Equal("customer_id", c.Principal.ID)}, Sort: []domain.Sort{{Field: "id", Desc: true}}})
	if e != nil {
		return domain.Result{}, e
	}
	latest, pending := map[string]domain.Record{}, map[string]domain.Record{}
	for _, claim := range claims {
		id := domain.Text(claim["gift_id"])
		if latest[id] == nil {
			latest[id] = claim
		}
		status := domain.Text(claim["status"])
		if (status == "waiting" || status == "approved") && pending[id] == nil {
			pending[id] = claim
		}
	}
	out := []domain.Record{}
	for _, g := range gifts {
		id := domain.Text(g["id"])
		o := domain.Pick(g, "id", "name", "total_point", "description")
		o["is_featured"] = domain.Bool(g["is_featured"])
		o["image"] = s.Asset(g["image"])
		o["image2"] = s.Asset(g["image2"])
		o["image3"] = s.Asset(g["image3"])
		o["claimable"] = total >= domain.Integer(g["total_point"]) && pending[id] == nil
		o["pending_claim"] = nil
		o["claim_status"] = nil
		if pending[id] != nil {
			o["pending_claim"] = domain.Pick(pending[id], "id", "status")
		}
		if claim := latest[id]; claim != nil {
			o["claim_status"] = map[string]string{"waiting": "Waiting", "approved": "Open", "rejected": "Closed", "completed": "Closed"}[domain.Text(claim["status"])]
		}
		out = append(out, o)
	}
	open, closed := 0, 0
	for _, claim := range latest {
		if domain.Text(claim["status"]) == "waiting" || domain.Text(claim["status"]) == "approved" {
			open++
		} else {
			closed++
		}
	}
	return domain.Success(domain.Record{"total_points": total, "gifts": out, "stats": domain.Record{"total_points": total, "open_count": open, "closed_count": closed}}, "Data gifts berhasil diambil"), nil
}
func (s *Service) claimGift(ctx context.Context, c domain.Command) (domain.Result, error) {
	if e := s.validate(ctx, c.Input, map[string]string{"gift_id": "required|string|exists:gifts,id", "notes": "nullable|string|max:500"}, ""); e != nil {
		return domain.Result{}, e
	}
	var claim, gift domain.Record
	e := s.Store.Transaction(ctx, func(db domain.Store) error {
		if e := s.lockCustomer(ctx, db, c.Principal.ID); e != nil {
			return e
		}
		var e error
		gift, e = db.Find(ctx, "gifts", domain.Q(domain.Equal("id", c.Input["gift_id"]), domain.Equal("status", "active")))
		if e != nil {
			return missing(e, "Gift tidak tersedia", "Gift tidak ditemukan atau tidak aktif")
		}
		existing, e := db.Count(ctx, "gift_claims", domain.Q(domain.Equal("customer_id", c.Principal.ID), domain.Equal("gift_id", gift["id"]), domain.In("status", []string{"waiting", "approved"})))
		if e != nil {
			return e
		}
		if existing > 0 {
			return domain.Fail("Sudah ada request claim", "Anda sudah memiliki request claim untuk gift ini")
		}
		rp, e := db.Find(ctx, "redeem_points", latestRedeem(c.Principal.ID))
		if e != nil {
			return missing(e, "Point tidak cukup", "Customer belum memiliki point")
		}
		balance := domain.Integer(rp["total_point_active"])
		required := domain.Integer(gift["total_point"])
		if required < 0 || balance < required {
			return domain.Fail("Point tidak cukup", fmt.Sprintf("Anda membutuhkan %d poin, saat ini hanya punya %d", required, balance))
		}
		claim, e = db.Insert(ctx, "gift_claims", domain.Record{"customer_id": c.Principal.ID, "gift_id": gift["id"], "redeem_point_id": rp["id"], "required_point": required, "customer_point_at_request": balance, "status": "waiting", "notes": c.Input["notes"]})
		if e != nil {
			return e
		}
		return s.audit(ctx, db, c.Principal, "gift.claim_requested", "gift_claim", claim["id"], "Claim gift "+domain.Text(gift["name"]))
	})
	if e != nil {
		return domain.Result{}, e
	}
	o := domain.Pick(claim, "id", "gift_id", "required_point", "status", "notes", "created_at")
	o["gift_name"] = gift["name"]
	o["created_at"] = dateISO(o["created_at"])
	return domain.Success(o, "Request claim berhasil dibuat, menunggu approval admin"), nil
}
func (s *Service) reviewGift(ctx context.Context, c domain.Command) (domain.Result, error) {
	rules := map[string]string{"admin_note": "nullable|string|max:500"}
	if c.Action == "reject-gift" {
		rules["admin_note"] = "required|string|max:500"
	}
	if e := s.validate(ctx, c.Input, rules, ""); e != nil {
		return domain.Result{}, e
	}
	initial, e := s.Find(ctx, "gift_claims", c.ID)
	if e != nil {
		return domain.Result{}, missing(e, "Claim tidak ditemukan", "Data tidak ditemukan")
	}
	var claim domain.Record
	e = s.Store.Transaction(ctx, func(db domain.Store) error {
		if e := s.lockCustomer(ctx, db, domain.Text(initial["customer_id"])); e != nil {
			return e
		}
		var e error
		claim, e = db.Find(ctx, "gift_claims", domain.Query{Filters: []domain.Filter{domain.Equal("id", c.ID)}, Lock: true})
		if e != nil {
			return e
		}
		expected := "waiting"
		if c.Action == "complete-gift" {
			expected = "approved"
		}
		if domain.Text(claim["status"]) != expected {
			return domain.Fail("Status tidak valid", fmt.Sprintf("Claim dengan status %s tidak bisa diproses (harus %s)", domain.Text(claim["status"]), expected))
		}
		changes := domain.Record{}
		switch c.Action {
		case "approve-gift":
			rp, e := db.Find(ctx, "redeem_points", latestRedeem(domain.Text(claim["customer_id"])))
			if e != nil {
				return missing(e, "Point customer tidak ditemukan", "Customer belum memiliki redeem point")
			}
			gift, e := db.Find(ctx, "gifts", idQuery(claim["gift_id"]))
			if e != nil {
				return e
			}
			desc := fmt.Sprintf("Claim gift \"%s\" (%d poin) - approved", domain.Text(gift["name"]), domain.Integer(claim["required_point"]))
			if domain.Text(c.Input["admin_note"]) != "" {
				desc += " | Note: " + domain.Text(c.Input["admin_note"])
			}
			amount := domain.Integer(claim["required_point"])
			if amount > 0 {
				if _, e = deduct(ctx, db, rp, amount, desc, true); e != nil {
					return e
				}
			}
			changes = domain.Record{"status": "approved", "approved_at": time.Now().UTC(), "approved_by": c.Principal.ID, "admin_note": c.Input["admin_note"]}
		case "reject-gift":
			changes = domain.Record{"status": "rejected", "approved_at": time.Now().UTC(), "approved_by": c.Principal.ID, "admin_note": c.Input["admin_note"]}
		case "complete-gift":
			changes = domain.Record{"status": "completed", "completed_at": time.Now().UTC()}
		}
		if e = db.Update(ctx, "gift_claims", idQuery(c.ID), changes); e != nil {
			return e
		}
		claim, e = db.Find(ctx, "gift_claims", idQuery(c.ID))
		if e != nil {
			return e
		}
		return s.audit(ctx, db, c.Principal, c.Action, "gift_claim", claim["id"], "Status claim: "+domain.Text(claim["status"]))
	})
	if e != nil {
		return domain.Result{}, e
	}
	view, e := s.claimView(ctx, claim)
	msg := map[string]string{"approve-gift": fmt.Sprintf("Claim approved, %d point telah dipotong dari customer", domain.Integer(claim["required_point"])), "reject-gift": "Claim ditolak, point customer tetap aman", "complete-gift": "Claim ditandai selesai (hadiah sudah diterima customer)"}[c.Action]
	return domain.Success(view, msg), e
}
