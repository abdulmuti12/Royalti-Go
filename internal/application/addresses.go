package application

import (
	"context"
	"fmt"
	"regexp"

	"github.com/abdulmuti12/api-point-go/internal/domain"
)

func (s *Service) addresses(ctx context.Context, c domain.Command) (domain.Result, error) {
	id, in := c.Principal.ID, c.Input
	if c.Action == "customer-profile" {
		row, e := s.Find(ctx, "customers", c.ID)
		if e != nil {
			return domain.Result{}, missing(e, "Not Found", "Customer tidak ditemukan")
		}
		return domain.Success(domain.Record{"general": domain.Pick(row, "id", "name", "full_name", "email", "phone_number", "status", "time", "created_at", "updated_at")}, "Success Load Data"), nil
	}
	if c.Action == "addresses" {
		rows, e := s.Store.List(ctx, "addresses", domain.Query{Filters: []domain.Filter{domain.Equal("user_id", id)}, Sort: []domain.Sort{{Field: "is_default", Desc: true}, {Field: "created_at", Desc: true}}})
		return domain.Success(rows, "Success Load Addresses"), e
	}
	if c.Action == "address-texts" {
		rows, e := s.Store.List(ctx, "address_text", domain.Query{Filters: []domain.Filter{domain.Equal("customer_id", id)}, Sort: []domain.Sort{{Field: "status", Desc: true}, {Field: "created_at", Desc: true}}})
		for i, r := range rows {
			rows[i] = domain.Pick(r, "id", "address", "note", "status")
		}
		return domain.Success(rows, "Success Load Address Texts"), e
	}
	table, owner := "address_text", "customer_id"
	if c.Action == "create-address" || c.Action == "delete-address" {
		table, owner = "addresses", "user_id"
	}
	rules := map[string]string{}
	switch c.Action {
	case "create-address":
		rules = map[string]string{"recipient_name": "required|string|max:255", "phone": "required|string|max:20", "address_line": "required|string", "province_id": "required|exists:master_province,prov_id", "province_name": "required|string|max:255", "city_id": "required|exists:master_city,city_id", "city_name": "required|string|max:255", "district_id": "required|exists:master_district,dis_id", "district_name": "required|string|max:255", "subdistrict_id": "required|exists:master_subdistrict,subdis_id", "subdistrict_name": "required|string|max:255", "postal_code": "required|string|max:10", "is_default": "sometimes|boolean"}
	case "create-address-text":
		rules = map[string]string{"address": "required|string", "note": "nullable|string|max:255"}
	case "update-address-text":
		rules = map[string]string{"id": "required", "address": "required|string", "note": "nullable|string|max:255"}
	default:
		rules = map[string]string{"id": "required"}
	}
	if e := s.validate(ctx, in, rules, ""); e != nil {
		return domain.Result{}, e
	}
	var result domain.Record
	e := s.Store.Transaction(ctx, func(db domain.Store) error {
		if e := s.lockCustomer(ctx, db, id); e != nil {
			return e
		}
		switch c.Action {
		case "create-address":
			data := domain.Pick(in, "recipient_name", "phone", "address_line", "province_id", "province_name", "city_id", "city_name", "district_id", "district_name", "subdistrict_id", "subdistrict_name", "postal_code")
			data["user_id"] = id
			data["is_default"] = domain.Bool(in["is_default"])
			for _, f := range []string{"recipient_name", "address_line", "province_name", "city_name", "district_name", "subdistrict_name"} {
				data[f] = sanitize(data[f])
			}
			data["phone"] = regexp.MustCompile(`[^0-9+\-\s]`).ReplaceAllString(domain.Text(data["phone"]), "")
			data["postal_code"] = regexp.MustCompile(`[^0-9]`).ReplaceAllString(domain.Text(data["postal_code"]), "")
			if domain.Bool(data["is_default"]) {
				if e := db.Update(ctx, table, domain.Q(domain.Equal(owner, id)), domain.Record{"is_default": false}); e != nil {
					return e
				}
			}
			var e error
			result, e = db.Insert(ctx, table, data)
			return e
		case "create-address-text":
			n, e := db.Count(ctx, table, domain.Q(domain.Equal(owner, id), domain.Equal("status", "active")))
			if e != nil {
				return e
			}
			status := "active"
			if n > 0 {
				status = "inactive"
			}
			result, e = db.Insert(ctx, table, domain.Record{"customer_id": id, "address": sanitize(in["address"]), "note": sanitize(in["note"]), "status": status})
			return e
		default:
			q := domain.Q(domain.Equal("id", in["id"]), domain.Equal(owner, id))
			row, e := db.Find(ctx, table, q)
			if e != nil {
				return missing(e, "Not Found", "Address not found or access denied")
			}
			if c.Action == "delete-address" || c.Action == "delete-address-text" {
				return db.Delete(ctx, table, q)
			}
			changes := domain.Record{}
			if c.Action == "toggle-address-text" {
				status := "inactive"
				if domain.Text(row["status"]) != "active" {
					status = "active"
					if e := db.Update(ctx, table, domain.Q(domain.Equal(owner, id), domain.Equal("status", "active")), domain.Record{"status": "inactive"}); e != nil {
						return e
					}
				}
				changes["status"] = status
			} else {
				changes["address"] = sanitize(in["address"])
				changes["note"] = sanitize(in["note"])
			}
			if e := db.Update(ctx, table, q, changes); e != nil {
				return e
			}
			result, e = db.Find(ctx, table, q)
			return e
		}
	})
	if e != nil {
		return domain.Result{}, e
	}
	message := map[string]string{"create-address": "Address created successfully", "delete-address": "Address deleted", "create-address-text": "Address text created successfully", "update-address-text": "Address text updated successfully", "toggle-address-text": "Address text status updated", "delete-address-text": "Address text deleted"}[c.Action]
	if message == "" {
		return domain.Result{}, fmt.Errorf("unknown address operation")
	}
	return domain.Success(result, message), nil
}
