package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/abdulmuti12/api-point-go/internal/domain"
)

type Service struct {
	Store               domain.Store
	Tokens              domain.Tokens
	Passwords           domain.Passwords
	Mail                domain.Mailer
	Google              domain.IdentityProvider
	AppURL, FrontendURL string
}

func (s *Service) Find(ctx context.Context, table, id string) (domain.Record, error) {
	return s.Store.Find(ctx, table, domain.Q(domain.Equal("id", id)))
}
func (s *Service) Optional(ctx context.Context, table string, q domain.Query) (domain.Record, error) {
	r, e := s.Store.Find(ctx, table, q)
	if errors.Is(e, domain.ErrNotFound) {
		return nil, nil
	}
	return r, e
}
func (s *Service) Asset(v any) any {
	path := domain.Text(v)
	if path == "" {
		return nil
	}
	return strings.TrimRight(s.AppURL, "/") + "/storage/" + strings.TrimPrefix(strings.TrimLeft(path, "/"), "storage/")
}
func (s *Service) page(ctx context.Context, t string, q domain.Query, in domain.Record, style string, defaultSize int) (domain.Page, error) {
	page := int(domain.Integer(in["page"]))
	if page < 1 {
		page = 1
	}
	size := defaultSize
	if (defaultSize != 10 || style == "order") && domain.Integer(in["per_page"]) > 0 {
		size = int(domain.Integer(in["per_page"]))
	}
	if size > 100 {
		size = 100
	}
	n, e := s.Store.Count(ctx, t, q)
	if e != nil {
		return domain.Page{}, e
	}
	q.Limit = size
	q.Offset = (page - 1) * size
	rows, e := s.Store.List(ctx, t, q)
	return domain.Page{Items: rows, Total: n, Number: page, Size: size, Style: style}, e
}
func idQuery(id any) domain.Query       { return domain.Q(domain.Equal("id", id)) }
func customerQuery(id any) domain.Query { return domain.Q(domain.Equal("customer_id", id)) }
func missing(e error, label, message string) error {
	if errors.Is(e, domain.ErrNotFound) {
		return domain.Fail(label, message)
	}
	return e
}
func secure(r domain.Record) domain.Record {
	o := domain.Copy(r)
	delete(o, "password")
	delete(o, "remember_token")
	return o
}

func dateISO(v any) any {
	if v == nil {
		return nil
	}
	t, e := time.Parse(time.RFC3339Nano, domain.Text(v))
	if e != nil {
		t, e = time.Parse("2006-01-02 15:04:05", domain.Text(v))
	}
	if e != nil {
		return v
	}
	return t.UTC().Format("2006-01-02T15:04:05-07:00")
}
func listIDs(rows []domain.Record, field string) []any {
	ids := []any{}
	for _, r := range rows {
		ids = append(ids, r[field])
	}
	return ids
}
func (s *Service) relatedCustomer(ctx context.Context, q domain.Query, keyword string) (domain.Query, error) {
	if keyword == "" {
		return q, nil
	}
	rows, e := s.Store.List(ctx, "customers", domain.Query{AnyLike: []string{"name", "full_name", "email"}, Keyword: keyword})
	if e != nil {
		return q, e
	}
	q.Filters = append(q.Filters, domain.In("customer_id", listIDs(rows, "id")))
	return q, nil
}
func (s *Service) lockCustomer(ctx context.Context, db domain.Store, id string) error {
	_, e := db.Find(ctx, "customers", domain.Query{Filters: []domain.Filter{domain.Equal("id", id)}, Lock: true})
	return e
}
func (s *Service) audit(ctx context.Context, db domain.Store, p domain.Principal, action, entity string, id any, description string) error {
	_, e := db.Insert(ctx, "audit_logs", domain.Record{"user_id": p.ID, "user_type": p.Kind, "user_name": p.Name, "action": action, "entity_type": entity, "entity_id": domain.Text(id), "description": description})
	return e
}

func (s *Service) Execute(ctx context.Context, c domain.Command) (domain.Result, error) {
	switch c.Action {
	case "login", "register", "social-register", "social-login", "set-password", "resend-verification", "verify-email", "me", "logout", "change-password", "forgot-password", "reset-password":
		return s.auth(ctx, c)
	case "index", "show", "edit", "create", "update", "delete", "role-edit", "set-active":
		return s.catalog(ctx, c)
	case "colors", "types", "lookup", "home-banner", "customer-promotion", "dashboard", "users", "settings", "subscribe", "contact":
		return s.public(ctx, c)
	case "addresses", "create-address", "delete-address", "address-texts", "create-address-text", "toggle-address-text", "update-address-text", "delete-address-text", "customer-profile":
		return s.addresses(ctx, c)
	case "orders", "order-detail", "create-order", "update-order", "delete-order", "customer-orders", "customer-order-detail", "customer-options":
		return s.orders(ctx, c)
	case "customer-transactions", "customer-transaction", "create-transaction", "update-transaction", "delete-transaction":
		return s.transactions(ctx, c)
	case "redeems", "redeem-detail", "calculate", "claim-points", "point-history", "member-level", "gifts", "claim-gift", "gift-history", "gift-claims", "gift-claim", "approve-gift", "reject-gift", "complete-gift":
		return s.loyalty(ctx, c)
	default:
		return domain.Result{}, fmt.Errorf("unknown use case %q", c.Action)
	}
}
