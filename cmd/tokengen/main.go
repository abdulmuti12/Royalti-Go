package main

import (
	"fmt"
	"log"
	"os"

	"github.com/abdulmuti12/api-point-go/internal/domain"
	"github.com/abdulmuti12/api-point-go/internal/platform"
)

func main() {
	c, e := platform.LoadConfig()
	if e != nil {
		log.Fatal(e)
	}
	sub := os.Args[1]
	kind := "customer"
	if len(os.Args) > 2 {
		kind = os.Args[2]
	}
	tok, e := platform.Tokens{Secret: c.Secret, Issuer: c.AppURL, TTL: c.TokenTTL}.Issue(domain.Principal{ID: sub, Kind: kind})
	if e != nil {
		log.Fatal(e)
	}
	fmt.Print(tok)
}
