package platform

import (
	"crypto/sha1"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/abdulmuti12/api-point-go/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func TestPHPBcryptCompatibility(t *testing.T) {
	hash, e := bcrypt.GenerateFromPassword([]byte("existing-password"), 4)
	if e != nil {
		t.Fatal(e)
	}
	phpHash := strings.Replace(string(hash), "$2a$", "$2y$", 1)
	if !(Passwords{}).Check("existing-password", phpHash) {
		t.Fatal("Laravel bcrypt hash rejected")
	}
	if (Passwords{}).Check("wrong", phpHash) {
		t.Fatal("wrong password accepted")
	}
}
func TestJWTRejectsTamperingExpiryAlgorithmAndWrongGuard(t *testing.T) {
	tokens := Tokens{Secret: strings.Repeat("s", 64), Issuer: "test", TTL: time.Hour}
	valid, e := tokens.Issue(domain.Principal{ID: "customer-uuid", Kind: "customer"})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = tokens.Parse(valid, "customer"); e != nil {
		t.Fatal(e)
	}
	if _, e = tokens.Parse(valid, "admin"); e == nil {
		t.Fatal("wrong guard accepted")
	}
	parts := strings.Split(valid, ".")
	parts[1] = "eyJzdWIiOiIxIn0"
	if _, e = tokens.Parse(strings.Join(parts, "."), "customer"); e == nil {
		t.Fatal("tampered token accepted")
	}
	for _, method := range []jwt.SigningMethod{jwt.SigningMethodHS256, jwt.SigningMethodHS384} {
		raw, e := jwt.NewWithClaims(method, jwt.MapClaims{"sub": "1", "kind": "customer", "jti": "test", "exp": time.Now().Add(-time.Minute).Unix()}).SignedString([]byte(tokens.Secret))
		if e != nil {
			t.Fatal(e)
		}
		if _, e = tokens.Parse(raw, "customer"); e == nil {
			t.Fatal("expired or wrong algorithm token accepted")
		}
	}
}
func TestLegacyTokensRequireOptInAndCorrectProvider(t *testing.T) {
	tokens := Tokens{Secret: strings.Repeat("s", 64), TTL: time.Hour}
	h := sha1.Sum([]byte("App\\Models\\Customer"))
	raw, e := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "uuid", "jti": "legacy", "exp": time.Now().Add(time.Hour).Unix(), "prv": hex.EncodeToString(h[:])}).SignedString([]byte(tokens.Secret))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = tokens.Parse(raw, "customer"); e == nil {
		t.Fatal("legacy token accepted without opt-in")
	}
	tokens.AcceptLegacy = true
	if _, e = tokens.Parse(raw, "customer"); e != nil {
		t.Fatal(e)
	}
	if _, e = tokens.Parse(raw, "admin"); e == nil {
		t.Fatal("wrong legacy provider accepted")
	}
}
