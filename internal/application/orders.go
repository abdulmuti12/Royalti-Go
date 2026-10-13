package application

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/abdulmuti12/api-point-go/internal/domain"
)

func (s *Service) activeRule(ctx context.Context) (domain.Record, error) {
	return s.Optional(ctx, "points", domain.Q(domain.Equal("status", "active")))
}
func earned(total int64, rule domain.Record) int64 {
	unit := domain.Integer(rule["range_point"])
	if rule == nil || unit <= 0 {
		return 0
	}
	return (total / unit) * domain.Integer(rule["point"])
}
func successful(status any) bool {
	return domain.Text(status) == "success" || domain.Text(status) == "delivered"
}

// Laravel explicitly casts this UUID field to an integer in the admin order
// views. Keep that legacy JSON type here; the UUID remains available in the
// customer endpoints. This adapter can change only with an FE contract update.
func legacyOrderCustomerID(v any) int64 {
	s := domain.Text(v)
	end := 0
	for end < len(s) && s[end] >= '0' && s[end] <= '9' {
		end++
	}
	if end == 0 {
		return 0
	}
	n, _ := strconv.ParseInt(s[:end], 10, 64)
	return n
}
func (s *Service) customerTotal(ctx context.Context, db domain.Store, id any) (int64, error) {
	rows, e := db.List(ctx, "orders", domain.Q(domain.Equal("customer_id", id), domain.In("status", []string{"delivered", "success"})))
	if e != nil {
		return 0, e
	}
	return sum(rows, "grand_total"), nil
}
func sum(rows []domain.Record, field string) int64 {
	var total int64
	for _, r := range rows {
		total += domain.Integer(r[field])
	}
	return total
}

func (s *Service) orderView(ctx context.Context, row domain.Record, detail, customer bool) (domain.Record, error) {
	items, e := s.Store.List(ctx, "order_items", domain.Q(domain.Equal("order_id", row["id"])))
	if e != nil {
		return nil, e
	}
	rule, e := s.activeRule(ctx)
	if e != nil {
		return nil, e
	}
	qty := sum(items, "quantity")
	if customer {
		pts := int64(0)
		if successful(row["status"]) {
			pts = earned(domain.Integer(row["grand_total"]), rule)
		}
		formatted := []domain.Record{}
		for _, i := range items {
			o := domain.Pick(i, "product_id", "product_name", "variant_name", "quantity", "price", "image")
			o["product_id"] = domain.Integer(i["product_id"])
			formatted = append(formatted, o)
		}
		o := domain.Record{"id": row["id"], "order_number": row["order_number"], "date": dateISO(row["created_at"]), "status": row["status"], "total": domain.Integer(row["grand_total"]), "total_items": qty, "point_earned": pts, "point_qualifies": pts > 0, "items": formatted}
		if detail {
			o["shipping"] = domain.Record{"recipient_name": row["recipient_name"], "phone": row["phone"], "address_line": row["shipping_address"], "province": row["province_name"], "city": row["city_name"], "district": row["district_name"], "subdistrict": row["subdistrict_name"], "postal_code": row["postal_code"]}
			o["notes"] = row["notes"]
			o["subtotal"] = domain.Integer(row["subtotal"])
			o["shipping_cost"] = domain.Integer(row["shipping_cost"])
		}
		return o, nil
	}
	user, e := s.Find(ctx, "customers", domain.Text(row["customer_id"]))
	if e != nil {
		return nil, e
	}
	total, e := s.customerTotal(ctx, s.Store, row["customer_id"])
	if e != nil {
		return nil, e
	}
	before := total
	if successful(row["status"]) {
		before -= domain.Integer(row["grand_total"])
	}
	delta := earned(total, rule) - earned(before, rule)
	if delta < 0 {
		delta = 0
	}
	var ruleView any
	recorded := int64(0)
	if rule != nil {
		ruleView = domain.Pick(rule, "name", "range_point", "point")
		rp, e := s.Optional(ctx, "redeem_points", domain.Q(domain.Equal("customer_id", row["customer_id"]), domain.Equal("point_id", rule["id"])))
		if e != nil {
			return nil, e
		}
		if rp != nil {
			recorded = domain.Integer(rp["total_point_earned"])
		}
	}
	o := domain.Record{"id": row["id"], "order_number": row["order_number"], "date": dateISO(row["created_at"]), "status": row["status"], "customer": domain.Pick(user, "id", "name", "full_name", "email", "phone_number"), "customer_total_transaction": total, "customer_points": earned(total, rule), "order_points_delta": delta, "recorded_points": recorded, "point_rule": ruleView}
	o["customer"].(domain.Record)["id"] = legacyOrderCustomerID(user["id"])
	if !detail {
		o["subtotal"] = domain.Integer(row["subtotal"])
		o["shipping_cost"] = domain.Integer(row["shipping_cost"])
		o["grand_total"] = domain.Integer(row["grand_total"])
		o["total_items"] = qty
		o["item_count"] = len(items)
		o["first_item"] = nil
		if len(items) > 0 {
			o["first_item"] = domain.Pick(items[0], "product_name", "variant_name", "image")
		}
		return o, nil
	}
	formatted := []domain.Record{}
	for _, i := range items {
		item := domain.Pick(i, "id", "product_id", "product_name", "name", "variant_name", "quantity", "price", "image")
		item["product_id"] = domain.Integer(i["product_id"])
		if item["name"] == nil {
			item["name"] = item["product_name"]
		}
		item["subtotal"] = domain.Integer(i["quantity"]) * domain.Integer(i["price"])
		for field, table := range map[string]string{"category": "category", "brand": "brands"} {
			related, e := s.Optional(ctx, table, idQuery(i[field+"_id"]))
			if e != nil {
				return nil, e
			}
			item[field] = nil
			if related != nil {
				item[field] = domain.Pick(related, "id", "name")
			}
		}
		formatted = append(formatted, item)
	}
	o["customer"].(domain.Record)["avatar"] = user["avatar"]
	o["notes"] = row["notes"]
	o["shipping"] = domain.Pick(row, "recipient_name", "phone", "shipping_address", "province_id", "province_name", "city_id", "city_name", "district_id", "district_name", "subdistrict_id", "subdistrict_name", "postal_code")
	o["items"] = formatted
	o["totals"] = domain.Record{"subtotal": domain.Integer(row["subtotal"]), "shipping_cost": domain.Integer(row["shipping_cost"]), "grand_total": domain.Integer(row["grand_total"]), "total_items": qty, "item_count": len(items)}
	trx, e := s.Optional(ctx, "transaction", domain.Q(domain.Equal("order_id", row["id"])))
	if e != nil {
		return nil, e
	}
	o["transaction"] = nil
	if trx != nil {
		o["transaction"] = domain.Pick(trx, "id", "transaction_code", "total_payment", "status", "note")
	}
	return o, nil
}

func (s *Service) orders(ctx context.Context, c domain.Command) (domain.Result, error) {
	switch c.Action {
	case "customer-options":
		q := domain.Query{Sort: []domain.Sort{{Field: "name"}}, AnyLike: []string{"name", "full_name", "email", "phone_number"}, Keyword: domain.Text(c.Input["keyword"])}
		page, e := s.page(ctx, "customers", q, c.Input, "paginator", 50)
		for i, r := range page.Items {
			page.Items[i] = domain.Pick(r, "id", "name", "full_name", "email", "phone_number", "status")
		}
		return domain.Success(page, "Data customer berhasil diambil"), e
	case "orders", "customer-orders":
		q := domain.Query{Sort: []domain.Sort{{Field: "id", Desc: true}}}
		if c.Action == "customer-orders" {
			q.Filters = append(q.Filters, domain.Equal("customer_id", c.Principal.ID))
			q.Sort = []domain.Sort{{Field: "created_at", Desc: true}}
			rows, e := s.Store.List(ctx, "orders", q)
			if e != nil {
				return domain.Result{}, e
			}
			out := []domain.Record{}
			for _, r := range rows {
				view, e := s.orderView(ctx, r, false, true)
				if e != nil {
					return domain.Result{}, e
				}
				out = append(out, view)
			}
			return domain.Success(out, "Data order berhasil diambil"), nil
		}
		if domain.Text(c.Input["order_number"]) != "" {
			q.Filters = append(q.Filters, domain.Filter{Field: "order_number", Op: "LIKE", Value: "%" + domain.Text(c.Input["order_number"]) + "%"})
		}
		if domain.Text(c.Input["status"]) != "" {
			q.Filters = append(q.Filters, domain.Equal("status", c.Input["status"]))
		}
		q, e := s.relatedCustomer(ctx, q, domain.Text(c.Input["customer_name"]))
		if e != nil {
			return domain.Result{}, e
		}
		page, e := s.page(ctx, "orders", q, c.Input, "order", 10)
		if e != nil {
			return domain.Result{}, e
		}
		for i, r := range page.Items {
			page.Items[i], e = s.orderView(ctx, r, false, false)
			if e != nil {
				return domain.Result{}, e
			}
		}
		return domain.Success(page, "Data order berhasil diambil"), nil
	case "order-detail", "customer-order-detail":
		q := idQuery(c.ID)
		customer := c.Action == "customer-order-detail"
		if customer {
			q.Filters = append(q.Filters, domain.Equal("customer_id", c.Principal.ID))
		}
		row, e := s.Store.Find(ctx, "orders", q)
		if e != nil {
			return domain.Result{}, missing(e, "Not Found", "Order tidak ditemukan")
		}
		view, e := s.orderView(ctx, row, true, customer)
		return domain.Success(view, "Detail order berhasil diambil"), e
	case "create-order", "update-order":
		rules := map[string]string{"customer_id": "required|exists:customers,id", "grand_total": "required|integer|min:0", "order_number": "nullable|string|max:50|unique:orders,order_number", "status": "nullable|in:processing,shipped,delivered,success,cancelled", "recipient_name": "nullable|string|max:255", "phone": "nullable|string|max:20", "shipping_address": "nullable|string", "province_id": "nullable|exists:master_province,prov_id", "city_id": "nullable|exists:master_city,city_id", "district_id": "nullable|exists:master_district,dis_id", "subdistrict_id": "nullable|exists:master_subdistrict,subdis_id", "postal_code": "nullable|string|max:10", "subtotal": "nullable|integer|min:0", "shipping_cost": "nullable|integer|min:0", "notes": "nullable|string"}
		if e := s.validate(ctx, c.Input, rules, c.ID); e != nil {
			return domain.Result{}, e
		}
		rawItems := slice(c.Input["items"])
		if c.Input["items"] != nil {
			if _, ok := c.Input["items"].([]any); !ok {
				return domain.Result{}, domain.Fail("Validasi item gagal", "items harus berupa array")
			}
		}
		for _, raw := range rawItems {
			item, ok := raw.(map[string]any)
			if !ok {
				if typed, ok := raw.(domain.Record); ok {
					item = typed
				} else {
					return domain.Result{}, domain.Fail("Validasi item gagal", "Item harus berupa object")
				}
			}
			if e := s.validate(ctx, domain.Record(item), map[string]string{"name": "required|string|max:255", "quantity": "required|integer|min:1", "price": "required|integer|min:0", "category_id": "nullable|exists:category,id", "brand_id": "nullable|exists:brands,id"}, ""); e != nil {
				return domain.Result{}, e
			}
		}
		data := domain.Record{}
		for field := range rules {
			if value, ok := c.Input[field]; ok {
				data[field] = value
			}
		}
		if c.Action == "create-order" {
			if domain.Text(data["order_number"]) == "" {
				token, e := domain.RandomToken()
				if e != nil {
					return domain.Result{}, e
				}
				data["order_number"] = "ORD-" + time.Now().Format("20060102") + "-" + token[:6]
			}
			if domain.Text(data["status"]) == "" {
				data["status"] = "processing"
			}
			if data["subtotal"] == nil {
				data["subtotal"] = data["grand_total"]
			}
			if data["shipping_cost"] == nil {
				data["shipping_cost"] = 0
			}
		}
		if domain.Text(data["recipient_name"]) == "" {
			user, e := s.Find(ctx, "customers", domain.Text(data["customer_id"]))
			if e != nil {
				return domain.Result{}, e
			}
			data["recipient_name"] = user["full_name"]
		}
		for _, loc := range []struct{ Key, Table, PK, Name string }{{"province", "master_province", "prov_id", "prov_name"}, {"city", "master_city", "city_id", "city_name"}, {"district", "master_district", "dis_id", "dis_name"}, {"subdistrict", "master_subdistrict", "subdis_id", "subdis_name"}} {
			if domain.Text(data[loc.Key+"_id"]) != "" {
				r, e := s.Store.Find(ctx, loc.Table, domain.Q(domain.Equal(loc.PK, data[loc.Key+"_id"])))
				if e != nil {
					return domain.Result{}, e
				}
				data[loc.Key+"_name"] = r[loc.Name]
			}
		}
		var row domain.Record
		e := s.Store.Transaction(ctx, func(db domain.Store) error {
			if e := s.lockCustomer(ctx, db, domain.Text(data["customer_id"])); e != nil {
				return missing(e, "Customer tidak ditemukan", "Data tidak ditemukan")
			}
			var e error
			if c.Action == "create-order" {
				row, e = db.Insert(ctx, "orders", data)
			} else {
				if _, e = db.Find(ctx, "orders", idQuery(c.ID)); e != nil {
					return e
				}
				e = db.Update(ctx, "orders", idQuery(c.ID), data)
				if e == nil {
					row, e = db.Find(ctx, "orders", idQuery(c.ID))
				}
			}
			if e != nil {
				return e
			}
			if c.Input["items"] != nil {
				if e = db.Delete(ctx, "order_items", domain.Q(domain.Equal("order_id", row["id"]))); e != nil {
					return e
				}
				for _, raw := range rawItems {
					item := domain.Record{}
					switch r := raw.(type) {
					case domain.Record:
						item = r
					case map[string]any:
						item = domain.Record(r)
					}
					i := domain.Pick(item, "name", "variant_name", "quantity", "price", "image", "category_id", "brand_id")
					i["product_name"] = item["name"]
					i["order_id"] = row["id"]
					if _, e = db.Insert(ctx, "order_items", i); e != nil {
						return e
					}
				}
			}
			return s.audit(ctx, db, c.Principal, "order."+c.Action, "order", row["id"], "Order "+domain.Text(row["order_number"])+" disimpan")
		})
		msg := "Transaksi berhasil ditambahkan"
		if c.Action == "update-order" {
			msg = "Order berhasil diperbarui"
		}
		return domain.Success(row, msg), e
	case "delete-order":
		e := s.Store.Transaction(ctx, func(db domain.Store) error {
			if _, e := db.Find(ctx, "orders", idQuery(c.ID)); e != nil {
				return e
			}
			for _, t := range []string{"order_items", "transaction"} {
				if e := db.Delete(ctx, t, domain.Q(domain.Equal("order_id", c.ID))); e != nil {
					return e
				}
			}
			return db.Delete(ctx, "orders", idQuery(c.ID))
		})
		return domain.Success(nil, "Order berhasil dihapus"), e
	}
	return domain.Result{}, fmt.Errorf("unknown order operation")
}
