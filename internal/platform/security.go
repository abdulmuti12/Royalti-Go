package platform

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/abdulmuti12/api-point-go/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Passwords struct{}

func (Passwords) Hash(s string) (string, error) {
	b, e := bcrypt.GenerateFromPassword([]byte(s), 12)
	return string(b), e
}
func (Passwords) Check(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(strings.Replace(hash, "$2y$", "$2a$", 1)), []byte(password)) == nil
}

type Tokens struct {
	Secret, Issuer string
	TTL            time.Duration
	AcceptLegacy   bool
}

func providerHash(kind string) string {
	model := "Admin"
	if kind == "customer" {
		model = "Customer"
	}
	h := sha1.Sum([]byte("App\\Models\\" + model))
	return hex.EncodeToString(h[:])
}
func (t Tokens) Issue(p domain.Principal) (string, error) {
	if p.Kind != "admin" && p.Kind != "customer" {
		return "", fmt.Errorf("invalid token kind")
	}
	id, e := domain.RandomToken()
	if e != nil {
		return "", e
	}
	now := time.Now()
	claims := jwt.MapClaims{"iss": t.Issuer, "iat": now.Unix(), "nbf": now.Unix(), "exp": now.Add(t.TTL).Unix(), "sub": p.ID, "jti": id, "kind": p.Kind, "prv": providerHash(p.Kind)}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(t.Secret))
}
func (t Tokens) Parse(raw, kind string) (domain.Principal, error) {
	var p domain.Principal
	token, e := jwt.Parse(raw, func(token *jwt.Token) (any, error) { return []byte(t.Secret), nil }, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if e != nil || token == nil || !token.Valid {
		return p, fmt.Errorf("invalid or expired token")
	}
	c, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return p, fmt.Errorf("invalid claims")
	}
	if domain.Text(c["kind"]) != kind {
		if !t.AcceptLegacy || domain.Text(c["kind"]) != "" || domain.Text(c["prv"]) != providerHash(kind) {
			return p, fmt.Errorf("token belongs to another guard")
		}
	}
	p.ID = domain.Text(c["sub"])
	p.Kind = kind
	p.JTI = domain.Text(c["jti"])
	exp, e := c.GetExpirationTime()
	if e != nil || exp == nil || p.ID == "" || p.JTI == "" {
		return p, fmt.Errorf("missing token claims")
	}
	p.Expires = exp.Time
	return p, nil
}
