package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/abdulmuti12/api-point-go/internal/application"
	"github.com/abdulmuti12/api-point-go/internal/domain"
	"github.com/abdulmuti12/api-point-go/internal/platform"
	"github.com/abdulmuti12/api-point-go/internal/repository/gormrepo"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

type fakeMail struct {
	mu       sync.Mutex
	messages []string
}

func (m *fakeMail) Send(_ context.Context, _, _, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, body)
	return nil
}

type fakeGoogle struct{}

func (fakeGoogle) Verify(context.Context, string, string) (domain.GoogleIdentity, error) {
	return domain.GoogleIdentity{}, fmt.Errorf("invalid token")
}

type fixture struct {
	h                                                    http.Handler
	s                                                    *application.Service
	db                                                   domain.Store
	admin, customer, other, adminID, customerID, otherID string
	storage                                              string
}

func setup(t *testing.T) *fixture {
	t.Helper()
	dir := t.TempDir()
	cfg := platform.Config{Driver: "sqlite", DSN: filepath.Join(dir, "test.db"), AppURL: "http://api.test", Storage: filepath.Join(dir, "storage"), Origins: []string{"http://localhost:3000"}, TokenTTL: time.Hour}
	if dsn := os.Getenv("MYSQL_TEST_DSN"); dsn != "" {
		parsed, e := mysqlDriver.ParseDSN(dsn)
		if e != nil {
			t.Fatal(e)
		}
		if !strings.HasPrefix(parsed.DBName, "api_point_go_test") {
			t.Fatal("MYSQL_TEST_DSN database must start with api_point_go_test")
		}
		random, e := domain.RandomToken()
		if e != nil {
			t.Fatal(e)
		}
		database := "api_point_go_test_" + random[:12]
		parsed.DBName = ""
		parsed.ParseTime = true
		adminDB, e := platform.OpenDatabase(platform.Config{Driver: "mysql", DSN: parsed.FormatDSN()})
		if e != nil {
			t.Fatal(e)
		}
		if e = adminDB.Exec("CREATE DATABASE `" + database + "`").Error; e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() {
			if e := adminDB.Exec("DROP DATABASE `" + database + "`").Error; e != nil {
				t.Error(e)
			}
			sqlDB, e := adminDB.DB()
			if e == nil {
				sqlDB.Close()
			}
		})
		parsed.DBName = database
		cfg.Driver = "mysql"
		cfg.DSN = parsed.FormatDSN()
	}
	db, e := platform.OpenDatabase(cfg)
	if e != nil {
		t.Fatal(e)
	}
	if e = gormrepo.Migrate(db, false); e != nil {
		t.Fatal(e)
	}
	sqlDB, e := db.DB()
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { sqlDB.Close() })
	store := gormrepo.New(db)
	tokens := platform.Tokens{Secret: strings.Repeat("s", 64), Issuer: cfg.AppURL, TTL: time.Hour}
	svc := &application.Service{Store: store, Tokens: tokens, Passwords: platform.Passwords{}, Mail: &fakeMail{}, Google: fakeGoogle{}, AppURL: cfg.AppURL, FrontendURL: "http://localhost:3000"}
	f := &fixture{s: svc, db: store, storage: cfg.Storage}
	f.h = New(svc, cfg)
	hash, e := bcrypt.GenerateFromPassword([]byte("Password123"), 4)
	if e != nil {
		t.Fatal(e)
	}
	admin := f.insert(t, "admins", domain.Record{"name": "Admin", "email": "admin@test.com", "password": string(hash), "status": "active"})
	role := f.insert(t, "roles", domain.Record{"name": "Administrator"})
	f.insert(t, "admin_role", domain.Record{"admin_id": admin["id"], "role_id": role["id"]})
	for _, name := range []string{"Admins", "Roles", "Promotions", "Banners", "ContactUs", "customer", "Orders", "Transactions", "Brands", "Categories", "Points", "Gifts", "Users", "Settings"} {
		menu := f.insert(t, "menus", domain.Record{"name": name, "route": name, "url": "/" + name})
		f.insert(t, "role_menu", domain.Record{"role_id": role["id"], "menu_id": menu["id"]})
	}
	f.adminID = domain.Text(admin["id"])
	user := f.insert(t, "customers", domain.Record{"name": "Customer", "full_name": "Customer Test", "email": "customer@test.com", "phone_number": "08123456789", "password": string(hash), "status": "active", "verify": 1})
	other := f.insert(t, "customers", domain.Record{"name": "Other", "full_name": "Other Test", "email": "other@test.com", "phone_number": "08111111111", "password": string(hash), "status": "active", "verify": 1})
	f.customerID = domain.Text(user["id"])
	f.otherID = domain.Text(other["id"])
	f.admin, e = tokens.Issue(domain.Principal{ID: f.adminID, Kind: "admin"})
	if e != nil {
		t.Fatal(e)
	}
	f.customer, e = tokens.Issue(domain.Principal{ID: f.customerID, Kind: "customer"})
	if e != nil {
		t.Fatal(e)
	}
	f.other, e = tokens.Issue(domain.Principal{ID: f.otherID, Kind: "customer"})
	if e != nil {
		t.Fatal(e)
	}
	return f
}
func (f *fixture) insert(t *testing.T, table string, r domain.Record) domain.Record {
	t.Helper()
	out, e := f.db.Insert(context.Background(), table, r)
	if e != nil {
		t.Fatal(e)
	}
	return out
}
func (f *fixture) request(t *testing.T, method, path, token string, body any) (int, domain.Record) {
	t.Helper()
	b, e := json.Marshal(body)
	if e != nil {
		t.Fatal(e)
	}
	r := httptest.NewRequest(method, path, bytes.NewReader(b))
	r.Header.Set("Content-Type", "application/json")
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	f.h.ServeHTTP(w, r)
	out := domain.Record{}
	if strings.Contains(w.Header().Get("Content-Type"), "json") {
		d := json.NewDecoder(w.Body)
		d.UseNumber()
		if e = d.Decode(&out); e != nil {
			t.Fatalf("%s %s: %s (%v)", method, path, w.Body.String(), e)
		}
	}
	return w.Code, out
}
func obj(t *testing.T, v any) domain.Record {
	t.Helper()
	switch m := v.(type) {
	case domain.Record:
		return m
	case map[string]any:
		return domain.Record(m)
	}
	t.Fatalf("expected object, got %#v", v)
	return nil
}
func status(t *testing.T, got, want int, body domain.Record) {
	t.Helper()
	if got != want {
		t.Fatalf("status=%d want=%d response=%#v", got, want, body)
	}
}

func TestRouteInventoryAndDispatch(t *testing.T) {
	f := setup(t)
	data, e := os.ReadFile("../../../docs/route-contract.json")
	if e != nil {
		t.Fatal(e)
	}
	var snapshot struct {
		Routes []struct{ Method, Path, Guard string }
	}
	if e = json.Unmarshal(data, &snapshot); e != nil {
		t.Fatal(e)
	}
	if len(snapshot.Routes) != 181 || len(Routes) != len(snapshot.Routes) {
		t.Fatalf("route count: %d vs %d", len(snapshot.Routes), len(Routes))
	}
	seen := map[string]bool{}
	for _, r := range Routes {
		seen[r.Method+" "+r.Path] = true
	}
	for _, r := range snapshot.Routes {
		if !seen[r.Method+" "+r.Path] {
			t.Fatalf("missing %s %s", r.Method, r.Path)
		}
		path := strings.ReplaceAll(r.Path, "{id}", "nonexistent-id")
		token := ""
		if r.Guard == "admin" {
			token = f.admin
		}
		if r.Guard == "customer" {
			token = f.customer
		}
		code, body := f.request(t, r.Method, path, token, map[string]any{})
		if code == 404 || code >= 500 {
			if code == 404 && strings.Contains(path, "order") {
				continue
			}
			t.Fatalf("route not dispatching safely %s %s: %d %#v", r.Method, path, code, body)
		}
	}
}

func TestAuthenticationGuardsRBACAndLogout(t *testing.T) {
	f := setup(t)
	code, b := f.request(t, "POST", "/api/admins/login", "", domain.Record{"email": "admin@test.com", "password": "Password123"})
	status(t, code, 200, b)
	data := obj(t, b["data"])
	if data["name"] != "Admin" || data["token"] == nil || data["menus"] == nil || data["role"] == nil {
		t.Fatal(data)
	}
	if data["password"] != nil {
		t.Fatal("password leaked")
	}
	code, b = f.request(t, "GET", "/api/admins/me", f.customer, nil)
	status(t, code, 401, b)
	code, b = f.request(t, "GET", "/api/customers/me", f.admin, nil)
	status(t, code, 401, b)
	code, b = f.request(t, "POST", "/api/customers/logout", f.customer, nil)
	status(t, code, 200, b)
	code, b = f.request(t, "GET", "/api/customers/me", f.customer, nil)
	status(t, code, 401, b)
	newRole := f.insert(t, "roles", domain.Record{"name": "Restricted"})
	restricted := f.insert(t, "admins", domain.Record{"name": "Restricted", "email": "restricted@test.com"})
	f.insert(t, "admin_role", domain.Record{"admin_id": restricted["id"], "role_id": newRole["id"]})
	token, e := f.s.Tokens.Issue(domain.Principal{ID: domain.Text(restricted["id"]), Kind: "admin"})
	if e != nil {
		t.Fatal(e)
	}
	code, b = f.request(t, "GET", "/api/admins/gift", token, nil)
	status(t, code, 403, b)
}

func TestLegacyPaginationShapes(t *testing.T) {
	f := setup(t)
	f.insert(t, "category", domain.Record{"name": "Sofa", "image": "categories/sofa.png"})
	code, b := f.request(t, "GET", "/api/admins/category", f.admin, nil)
	status(t, code, 200, b)
	outer := obj(t, b["data"])
	nested := obj(t, outer["data"])
	items := nested["data"].([]any)
	if obj(t, items[0])["image"] != "http://api.test/storage/categories/sofa.png" {
		t.Fatal(items)
	}
	if obj(t, outer["meta"])["total"] == nil || obj(t, outer["links"])["first"] != "http://api.test/api/admins/category?page=1" {
		t.Fatal(outer)
	}
	code, b = f.request(t, "GET", "/api/admins/customer", f.admin, nil)
	status(t, code, 200, b)
	flat := obj(t, b["data"])
	if flat["current_page"] == nil || flat["first_page_url"] == nil || flat["data"] == nil {
		t.Fatal(flat)
	}
	code, b = f.request(t, "GET", "/api/admins/role", f.admin, nil)
	status(t, code, 200, b)
	if _, exists := b["success"]; exists {
		t.Fatal("role index must have no outer envelope")
	}
	code, b = f.request(t, "GET", "/api/admins/category?raw=false", f.admin, nil)
	status(t, code, 200, b)
	if _, ok := b["data"].([]any); !ok {
		t.Fatal("PHP raw=false is truthy")
	}
	code, b = f.request(t, "GET", "/api/admins/admin?raw=false", f.admin, nil)
	status(t, code, 200, b)
	if obj(t, b["data"])["current_page"] == nil {
		t.Fatal("admin raw uses FILTER_VALIDATE_BOOLEAN")
	}
}

func TestPointsCalculationIdempotencyAndConcurrentClaims(t *testing.T) {
	f := setup(t)
	f.insert(t, "points", domain.Record{"name": "Base", "status": "active", "price_point": 1000, "range_point": 100000, "point": 1})
	f.insert(t, "orders", domain.Record{"order_number": "ORD-1", "customer_id": f.customerID, "grand_total": 1075000, "subtotal": 1075000, "shipping_cost": 0, "status": "delivered"})
	path := "/api/admins/redeem-point/calculate/" + f.customerID
	code, b := f.request(t, "POST", path, f.admin, nil)
	status(t, code, 200, b)
	if domain.Integer(obj(t, b["data"])["total_point_active"]) != 10 {
		t.Fatal(b)
	}
	code, b = f.request(t, "POST", path, f.admin, nil)
	status(t, code, 200, b)
	hist, e := f.db.Count(context.Background(), "redeem_histories", domain.Q(domain.Equal("customer_id", f.customerID)))
	if e != nil || hist != 1 {
		t.Fatalf("duplicate earning: %d %v", hist, e)
	}
	var successes atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, e := f.s.Execute(context.Background(), domain.Command{Action: "claim-points", Input: domain.Record{"customer_id": f.customerID, "point_amount": 7}, Principal: domain.Principal{ID: f.adminID, Kind: "admin"}})
			if e == nil && res.Status == 200 {
				successes.Add(1)
			}
		}()
	}
	wg.Wait()
	if successes.Load() != 1 {
		t.Fatalf("successful claims=%d", successes.Load())
	}
	rp, e := f.db.Find(context.Background(), "redeem_points", domain.Q(domain.Equal("customer_id", f.customerID)))
	if e != nil {
		t.Fatal(e)
	}
	if domain.Integer(rp["total_point_active"]) != 3 || domain.Integer(rp["total_point_closed"]) != 7 {
		t.Fatal(rp)
	}
}

func TestGiftApprovalIsAppliedOnce(t *testing.T) {
	f := setup(t)
	p := f.insert(t, "points", domain.Record{"name": "Base", "status": "active", "range_point": 100000, "point": 1})
	f.insert(t, "redeem_points", domain.Record{"customer_id": f.customerID, "point_id": p["id"], "total_point_active": 20, "total_point_earned": 20, "total_point_closed": 0, "total_point_claimed": 0})
	gift := f.insert(t, "gifts", domain.Record{"name": "Gift", "status": "active", "total_point": 8})
	code, b := f.request(t, "POST", "/api/customers/gifts/claim", f.customer, domain.Record{"gift_id": gift["id"]})
	status(t, code, 200, b)
	claimID := domain.Text(obj(t, b["data"])["id"])
	code, b = f.request(t, "POST", "/api/customers/gifts/claim", f.customer, domain.Record{"gift_id": gift["id"]})
	status(t, code, 401, b)
	var success atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, e := f.s.Execute(context.Background(), domain.Command{Action: "approve-gift", ID: claimID, Input: domain.Record{}, Principal: domain.Principal{ID: f.adminID, Kind: "admin", Name: "Admin"}})
			if e == nil {
				success.Add(1)
			}
		}()
	}
	wg.Wait()
	if success.Load() != 1 {
		t.Fatalf("approved=%d", success.Load())
	}
	rp, e := f.db.Find(context.Background(), "redeem_points", domain.Q(domain.Equal("customer_id", f.customerID)))
	if e != nil {
		t.Fatal(e)
	}
	if domain.Integer(rp["total_point_active"]) != 12 || domain.Integer(rp["total_point_claimed"]) != 8 {
		t.Fatal(rp)
	}
	code, b = f.request(t, "POST", "/api/admins/gift-claims/"+claimID+"/complete", f.admin, nil)
	status(t, code, 200, b)
	code, b = f.request(t, "POST", "/api/admins/gift-claims/"+claimID+"/approve", f.admin, nil)
	status(t, code, 401, b)
}

func TestOwnershipAndMemberLevel(t *testing.T) {
	f := setup(t)
	order := f.insert(t, "orders", domain.Record{"order_number": "PRIVATE", "customer_id": f.customerID, "grand_total": 200, "status": "processing"})
	address := f.insert(t, "addresses", domain.Record{"user_id": f.customerID, "recipient_name": "Customer"})
	code, b := f.request(t, "GET", "/api/customers/get-orders/"+domain.Text(order["id"]), f.other, nil)
	status(t, code, 404, b)
	code, b = f.request(t, "POST", "/api/customers/delete-address", f.other, domain.Record{"id": address["id"]})
	status(t, code, 401, b)
	code, b = f.request(t, "POST", "/api/customers/transactions", f.other, domain.Record{"order_id": order["id"], "total_payment": 200, "status": "pending", "address_id": address["id"]})
	status(t, code, 401, b)
	code, b = f.request(t, "POST", "/api/customers/transactions", f.customer, domain.Record{"order_id": order["id"], "total_payment": 200, "status": "success", "address_id": address["id"]})
	status(t, code, 401, b)
	p := f.insert(t, "points", domain.Record{"name": "Rule", "status": "active"})
	f.insert(t, "redeem_points", domain.Record{"customer_id": f.customerID, "point_id": p["id"], "total_point_earned": 150, "total_point_active": 10})
	f.insert(t, "member_levels", domain.Record{"name": "Silver", "status": "active", "min_point": 0, "max_point": 199})
	f.insert(t, "member_levels", domain.Record{"name": "Gold", "status": "active", "min_point": 200, "max_point": 0})
	code, b = f.request(t, "GET", "/api/customers/member-level", f.customer, nil)
	status(t, code, 200, b)
	data := obj(t, b["data"])
	if domain.Integer(data["total_points"]) != 150 || obj(t, data["current_level"])["name"] != "Silver" || domain.Integer(obj(t, data["progress"])["percent"]) != 75 {
		t.Fatal(data)
	}
}

func TestMultipartMethodOverrideAndFileCleanup(t *testing.T) {
	f := setup(t)
	category := f.insert(t, "category", domain.Record{"name": "Old", "image": "categories/old.png"})
	old := filepath.Join(f.storage, "categories", "old.png")
	if e := os.MkdirAll(filepath.Dir(old), 0750); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(old, []byte("old"), 0640); e != nil {
		t.Fatal(e)
	}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	mw.WriteField("_method", "PUT")
	mw.WriteField("name", "New")
	fw, e := mw.CreateFormFile("image", "new.png")
	if e != nil {
		t.Fatal(e)
	}
	if e = png.Encode(fw, image.NewRGBA(image.Rect(0, 0, 2, 2))); e != nil {
		t.Fatal(e)
	}
	mw.Close()
	r := httptest.NewRequest("POST", "/api/admins/category/"+domain.Text(category["id"]), &body)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	r.Header.Set("Authorization", "Bearer "+f.admin)
	w := httptest.NewRecorder()
	f.h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if _, e = os.Stat(old); !os.IsNotExist(e) {
		t.Fatalf("old upload not deleted: %v", e)
	}
	updated, e := f.s.Find(context.Background(), "category", domain.Text(category["id"]))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = os.Stat(filepath.Join(f.storage, domain.Text(updated["image"]))); e != nil {
		t.Fatal(e)
	}
	if updated["name"] != "New" {
		t.Fatal(updated)
	}
}

func TestOriginAndLoginRateLimit(t *testing.T) {
	f := setup(t)
	r := httptest.NewRequest("POST", "/api/admins/login", strings.NewReader(`{"email":"admin@test.com","password":"Password123"}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", "https://evil.test")
	w := httptest.NewRecorder()
	f.h.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal(w.Code)
	}
	for i := 0; i < 5; i++ {
		code, b := f.request(t, "POST", "/api/admins/login", "", domain.Record{"email": "admin@test.com", "password": "wrong"})
		status(t, code, 401, b)
	}
	code, b := f.request(t, "POST", "/api/admins/login", "", domain.Record{"email": "admin@test.com", "password": "wrong"})
	status(t, code, 429, b)
}
