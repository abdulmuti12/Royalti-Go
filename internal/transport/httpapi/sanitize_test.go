package httpapi

import (
	"context"
	"strings"
	"testing"

	"github.com/abdulmuti12/api-point-go/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

func TestLegacyInputSanitizationAndMenuCollation(t *testing.T) {
	if got := sanitizeLegacy(`<script>alert(1)</script><p onclick="x">A & B</p>`); got != `&lt;p &gt;A &amp; B&lt;/p&gt;` {
		t.Fatal(got)
	}
	f := setup(t)
	hash, e := bcrypt.GenerateFromPassword([]byte("Pass&amp;Word123"), 4)
	if e != nil {
		t.Fatal(e)
	}
	if e = f.db.Update(context.Background(), "admins", domain.Q(domain.Equal("id", f.adminID)), domain.Record{"password": string(hash)}); e != nil {
		t.Fatal(e)
	}
	code, b := f.request(t, "POST", "/api/admins/login", "", domain.Record{"email": "admin@test.com", "password": "Pass&Word123"})
	status(t, code, 200, b)
	menus, e := f.db.List(context.Background(), "menus", domain.Query{})
	if e != nil {
		t.Fatal(e)
	}
	for _, menu := range menus {
		if e = f.db.Update(context.Background(), "menus", domain.Q(domain.Equal("id", menu["id"])), domain.Record{"route": strings.ToLower(domain.Text(menu["route"]))}); e != nil {
			t.Fatal(e)
		}
	}
	code, b = f.request(t, "GET", "/api/admins/point", f.admin, nil)
	status(t, code, 200, b)
}
