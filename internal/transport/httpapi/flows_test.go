package httpapi

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/abdulmuti12/api-point-go/internal/domain"
)

type failedMail struct{}

func (failedMail) Send(context.Context, string, string, string) error {
	return fmt.Errorf("mail delivery rejected")
}

func TestRegistrationFirstPasswordSetupAndSingleUseToken(t *testing.T) {
	f := setup(t)
	code, b := f.request(t, "POST", "/api/customers/register", "", domain.Record{"name": "New", "email": "new@test.com", "phone_number": "08222222222"})
	status(t, code, 200, b)
	m := f.s.Mail.(*fakeMail)
	m.mu.Lock()
	message := m.messages[0]
	m.mu.Unlock()
	password := strings.Split(strings.Split(message, "Password sementara: ")[1], "\n")[0]
	code, b = f.request(t, "POST", "/api/customers/login-customer", "", domain.Record{"email": "new@test.com", "password": password})
	status(t, code, 403, b)
	token := b["setup_token"]
	if b["needs_password_setup"] != true {
		t.Fatal(b)
	}
	input := domain.Record{"email": "new@test.com", "setup_token": token, "new_password": "NewPassword123", "confirm_password": "NewPassword123"}
	code, b = f.request(t, "POST", "/api/customers/set-password", "", input)
	status(t, code, 200, b)
	if obj(t, b["data"])["token"] == nil {
		t.Fatal(b)
	}
	if _, exists := b["status"]; exists {
		t.Fatal("set-password success has no status key in Laravel")
	}
	code, b = f.request(t, "POST", "/api/customers/set-password", "", input)
	status(t, code, 400, b)
	code, b = f.request(t, "POST", "/api/customers/login-customer", "", domain.Record{"email": "new@test.com", "password": "NewPassword123"})
	status(t, code, 200, b)
	code, b = f.request(t, "POST", "/api/customers/register", "", domain.Record{"name": "Duplicate", "email": "new@test.com", "phone_number": "08333333333"})
	status(t, code, 409, b)
}
func TestRegistrationRollsBackWhenMailFails(t *testing.T) {
	f := setup(t)
	f.s.Mail = failedMail{}
	_, e := f.s.Execute(context.Background(), domain.Command{Action: "register", Input: domain.Record{"name": "New", "email": "rollback@test.com", "phone_number": "08444444444"}})
	if e == nil {
		t.Fatal("failed mail ignored")
	}
	n, e := f.db.Count(context.Background(), "customers", domain.Q(domain.Equal("email", "rollback@test.com")))
	if e != nil || n != 0 {
		t.Fatalf("orphaned account: %d %v", n, e)
	}
}
func TestOrderItemsAndAdminCreationContract(t *testing.T) {
	f := setup(t)
	code, b := f.request(t, "POST", "/api/admins/order", f.admin, domain.Record{"customer_id": f.customerID, "grand_total": 1500, "status": "success", "items": []any{map[string]any{"name": "Sofa", "quantity": 2, "price": 750}}})
	status(t, code, 200, b)
	if b["message"] != "Transaksi berhasil ditambahkan" {
		t.Fatal(b)
	}
	id := obj(t, b["data"])["id"]
	code, b = f.request(t, "GET", "/api/admins/get-orders/"+domain.Text(id), f.admin, nil)
	status(t, code, 200, b)
	data := obj(t, b["data"])
	if domain.Integer(obj(t, data["totals"])["total_items"]) != 2 {
		t.Fatal(data)
	}
	if _, ok := obj(t, data["customer"])["id"].(interface{ String() string }); !ok {
		t.Fatal("admin order customer.id must preserve numeric Laravel cast")
	}
	role, e := f.db.Find(context.Background(), "roles", domain.Query{})
	if e != nil {
		t.Fatal(e)
	}
	code, b = f.request(t, "POST", "/api/admins/admin", f.admin, domain.Record{"name": "Added", "email": "added@test.com", "password": "StrongPassword123", "role_id": role["id"], "status": "active"})
	status(t, code, 200, b)
	if obj(t, b["data"])["password"] != nil {
		t.Fatal("password leaked")
	}
}
func TestAddressTextSelectionRemainsSingularUnderConcurrency(t *testing.T) {
	f := setup(t)
	a := f.insert(t, "address_text", domain.Record{"customer_id": f.customerID, "address": "A", "status": "inactive"})
	b := f.insert(t, "address_text", domain.Record{"customer_id": f.customerID, "address": "B", "status": "inactive"})
	var wg sync.WaitGroup
	for _, r := range []domain.Record{a, b} {
		wg.Add(1)
		go func(r domain.Record) {
			defer wg.Done()
			_, e := f.s.Execute(context.Background(), domain.Command{Action: "toggle-address-text", Input: domain.Record{"id": r["id"]}, Principal: domain.Principal{ID: f.customerID, Kind: "customer"}})
			if e != nil {
				t.Error(e)
			}
		}(r)
	}
	wg.Wait()
	n, e := f.db.Count(context.Background(), "address_text", domain.Q(domain.Equal("customer_id", f.customerID), domain.Equal("status", "active")))
	if e != nil || n != 1 {
		t.Fatalf("active addresses=%d error=%v", n, e)
	}
}
func TestParameterBindingAndInvalidPointRule(t *testing.T) {
	f := setup(t)
	f.insert(t, "category", domain.Record{"name": "Sofa"})
	code, b := f.request(t, "GET", "/api/admins/category?name=%27%20OR%201%3D1%20--", f.admin, nil)
	status(t, code, 200, b)
	if domain.Integer(obj(t, obj(t, b["data"])["meta"])["total"]) != 0 {
		t.Fatal("search query not parameterized")
	}
	code, b = f.request(t, "POST", "/api/admins/point", f.admin, domain.Record{"name": "Invalid", "price_point": 1000, "range_point": 0, "point": 1})
	status(t, code, 401, b)
	n, e := f.db.Count(context.Background(), "points", domain.Query{})
	if e != nil || n != 0 {
		t.Fatal("invalid point rule persisted")
	}
}
