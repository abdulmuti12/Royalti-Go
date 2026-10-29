// Command resettool generates a password-reset link for a customer email,
// printing the raw reset token and a ready-to-use reset-password payload.
//
// Usage:
//
//	go run ./cmd/resettool -email customer@example.com
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/abdulmuti12/api-point-go/internal/domain"
	"github.com/abdulmuti12/api-point-go/internal/platform"
	"github.com/abdulmuti12/api-point-go/internal/repository/gormrepo"
)

func main() {
	email := flag.String("email", "", "customer email to reset")
	ttl := flag.Duration("ttl", time.Hour, "reset token TTL")
	resetURL := flag.String("url", "http://localhost:3000/reset-password", "frontend reset URL (token + email appended as query params)")
	flag.Parse()
	if *email == "" {
		log.Fatal("-email is required")
	}

	c, e := platform.LoadConfig()
	if e != nil {
		log.Fatal(e)
	}
	db, e := platform.OpenDatabase(c)
	if e != nil {
		log.Fatal(e)
	}

	store := gormrepo.New(db)
	ctx := context.Background()
	n, e := store.Count(ctx, "customers", domain.Q(domain.Equal("email", *email)))
	if e != nil {
		log.Fatal(e)
	}
	if n == 0 {
		log.Fatalf("customer email %q not found in customers table", *email)
	}

	token, e := domain.RandomToken()
	if e != nil {
		log.Fatal(e)
	}
	h := sha256.Sum256([]byte(token))
	hash := hex.EncodeToString(h[:])
	key := "reset:" + *email

	// Replace any existing pending reset token for this email (same behaviour
	// as the API's forgot-password handler).
	if e := store.Delete(ctx, "point_go_tokens", domain.Q(domain.Equal("key", key))); e != nil {
		log.Fatal(e)
	}
	_, e = store.Insert(ctx, "point_go_tokens", domain.Record{
		"key":        key,
		"hash":       hash,
		"subject":    *email,
		"kind":       "reset",
		"expires_at": time.Now().UTC().Add(*ttl),
	})
	if e != nil {
		log.Fatal(e)
	}

	fmt.Println("reset token written to point_go_tokens.")
	fmt.Printf("token:        %s\n", token)
	fmt.Printf("expires_at:   %s\n", time.Now().UTC().Add(*ttl).Format(time.RFC3339))
	fmt.Println()
	fmt.Println("Reset via API (one-shot, token is consumed on success):")
	fmt.Printf("  curl -X POST http://localhost:8001/api/customers/reset-password \\\n       -H 'Content-Type: application/json' \\\n       -d '{\"token\":\"%s\",\"email\":\"%s\",\"password\":\"NewPass123\",\"password_confirmation\":\"NewPass123\"}'\n", token, *email)
	fmt.Println()
	fmt.Println("Or share this link with the user (frontend must read ?token=&email=):")
	fmt.Printf("  %s?token=%s&email=%s\n", *resetURL, token, *email)
}
