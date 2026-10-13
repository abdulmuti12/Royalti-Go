package application

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/abdulmuti12/api-point-go/internal/domain"
)

var tags = regexp.MustCompile(`<[^>]*>`)

func sanitize(v any) any {
	if v == nil {
		return nil
	}
	return tags.ReplaceAllString(domain.Text(v), "")
}

func (s *Service) public(ctx context.Context, c domain.Command) (domain.Result, error) {
	switch c.Action {
	case "colors":
		rows := []domain.Record{}
		for _, color := range strings.Split("Red,Green,Blue,Yellow,Cyan,Magenta,Black,White,Gray,Orange,Purple,Pink,Brown,Lime,Olive,Teal,Navy,Maroon,Silver,Gold,Beige", ",") {
			rows = append(rows, domain.Record{"color": color})
		}
		return domain.Success(rows, "Success Load Data"), nil
	case "types":
		return domain.Success([]string{"Custom Made", "Pre Order", "Limited Edition", "Sales Stock", "Ready Stock"}, "Success Load Data"), nil
	case "lookup":
		q := domain.Query{}
		msg := "Success Load Data"
		switch c.Table {
		case "master_city":
			q.Filters = append(q.Filters, domain.Equal("prov_id", c.Input["prov_id"]))
		case "master_district":
			q.Filters = append(q.Filters, domain.Equal("city_id", c.Input["city_id"]))
		case "master_subdistrict":
			q.Filters = append(q.Filters, domain.Equal("dis_id", c.Input["dis_id"]))
		case "master_postal_code":
			for _, f := range []string{"prov_id", "city_id", "dis_id", "subdis_id"} {
				q.Filters = append(q.Filters, domain.Equal(f, c.Input[f]))
			}
			row, e := s.Optional(ctx, c.Table, q)
			return domain.Success(row, msg), e
		case "active-logo":
			row, e := s.Optional(ctx, "logos", domain.Q(domain.Equal("is_active", 1)))
			if e != nil {
				return domain.Result{}, e
			}
			if row == nil {
				return domain.Result{}, domain.Fail("Tidak ada logo aktif", "Data not found")
			}
			r, e := s.resource(ctx, "logos", row)
			return domain.Success(r, "Logo aktif berhasil diambil"), e
		}
		if domain.Bool(c.Input["sorted"]) {
			q.Sort = []domain.Sort{{Field: "name"}}
		}
		rows, e := s.Store.List(ctx, c.Table, q)
		if e != nil {
			return domain.Result{}, e
		}
		if !strings.HasPrefix(c.Table, "master_") {
			for i, r := range rows {
				rows[i] = domain.Pick(r, "id", "name")
			}
		}
		if c.Table == "menus" {
			msg = "Data LoadSuccess"
		}
		if domain.Bool(c.Input["sorted"]) {
			if c.Table == "brands" {
				msg = "Data brand berhasil diambil"
			} else {
				msg = "Data category berhasil diambil"
			}
		}
		return domain.Success(rows, msg), nil
	case "home-banner":
		rows, e := s.Store.List(ctx, "banner", domain.Q(domain.Equal("status", "Active")))
		if e != nil {
			return domain.Result{}, e
		}
		slides := []domain.Record{}
		for _, row := range rows {
			for i := 1; i <= 5; i++ {
				f, n, title := "file", "note", "title"
				if i > 1 {
					f += fmt.Sprint(i)
					n += fmt.Sprint(i)
					title += fmt.Sprint(i)
				}
				if domain.Text(row[f]) == "" {
					continue
				}
				typ := row["type"]
				if typ == nil {
					typ = "main"
				}
				slides = append(slides, domain.Record{"src": s.Asset(row[f]), "alt": domain.Text(row["name"]), "type": typ, "title": domain.Text(row[title]), "subtitle": " ", "description": domain.Text(row[n]), "tag": ""})
			}
		}
		return domain.Success(slides, "Success Load Slides"), nil
	case "customer-promotion":
		row, e := s.Optional(ctx, "promotions", domain.Query{Filters: []domain.Filter{domain.Equal("status", "active")}, Sort: []domain.Sort{{Field: "id", Desc: true}}})
		if e != nil {
			return domain.Result{}, e
		}
		if row != nil {
			row = domain.Pick(row, "id", "name", "description", "file")
			row["file"] = s.Asset(row["file"])
		}
		return domain.Success(row, "Success Load Data"), nil
	case "dashboard":
		data := domain.Record{}
		for field, t := range map[string]string{"total_categories": "category", "total_brands": "brands", "total_customers": "customers", "total_admins": "admins", "total_role": "roles"} {
			n, e := s.Store.Count(ctx, t, domain.Query{})
			if e != nil {
				return domain.Result{}, e
			}
			data[field] = n
		}
		return domain.Success(data, "Dashboard data retrieved successfully"), nil
	case "users":
		return domain.Result{Status: 200, Body: domain.Record{"message": "Users Page"}}, nil
	case "settings":
		return domain.Result{Status: 200, Body: domain.Record{"message": "Settings Page"}}, nil
	case "subscribe":
		if e := s.validate(ctx, c.Input, map[string]string{"email": "required|email|unique:email_subscribers,email"}, ""); e != nil {
			var fail *domain.Failure
			if errors.As(e, &fail) {
				return domain.Result{}, domain.Reject(400, domain.Record{"error": fail.Message})
			}
			return domain.Result{}, e
		}
		row, e := s.Store.Insert(ctx, "email_subscribers", domain.Pick(c.Input, "email"))
		return domain.Result{Status: 201, Body: domain.Record{"status": true, "message": "Email Sudah Ada", "data": row}}, e
	case "contact":
		if e := s.validate(ctx, c.Input, map[string]string{"name": "required|string|max:255", "email": "required|email|max:255", "phone_number": "nullable|string|max:20", "description": "nullable|string|max:1000"}, ""); e != nil {
			return domain.Result{}, e
		}
		data := domain.Pick(c.Input, "name", "email", "phone_number", "description")
		for _, field := range []string{"name", "phone_number", "description"} {
			data[field] = sanitize(data[field])
		}
		row, e := s.Store.Insert(ctx, "contact_us", data)
		return domain.Success(row, "Add Success"), e
	}
	return domain.Result{}, fmt.Errorf("unknown public operation")
}
