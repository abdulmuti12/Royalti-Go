package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/abdulmuti12/api-point-go/internal/domain"
	"github.com/abdulmuti12/api-point-go/internal/platform"
	"github.com/abdulmuti12/api-point-go/internal/repository/gormrepo"
)

func main() {
	email := flag.String("email", "", "initial admin email")
	name := flag.String("name", "Administrator", "initial admin name")
	flag.Parse()
	password := os.Getenv("SEED_ADMIN_PASSWORD")
	if *email == "" || len(password) < 8 || len(password) > 72 {
		log.Fatal("set --email and SEED_ADMIN_PASSWORD (8 to 72 bytes)")
	}
	c, e := platform.LoadConfig()
	if e != nil {
		log.Fatal(e)
	}
	db, e := platform.OpenDatabase(c)
	if e != nil {
		log.Fatal(e)
	}
	sqlDB, e := db.DB()
	if e != nil {
		log.Fatal(e)
	}
	defer sqlDB.Close()
	store := gormrepo.New(db)
	hash, e := (platform.Passwords{}).Hash(password)
	if e != nil {
		log.Fatal(e)
	}
	e = store.Transaction(context.Background(), func(tx domain.Store) error {
		ctx := context.Background()
		n, e := tx.Count(ctx, "admins", domain.Query{})
		if e != nil {
			return e
		}
		if n > 0 {
			return domain.Fail("Seed refused", "an admin already exists")
		}
		role, e := tx.Insert(ctx, "roles", domain.Record{"name": "Administrator"})
		if e != nil {
			return e
		}
		admin, e := tx.Insert(ctx, "admins", domain.Record{"name": *name, "email": *email, "password": hash, "status": "active"})
		if e != nil {
			return e
		}
		if _, e = tx.Insert(ctx, "admin_role", domain.Record{"admin_id": admin["id"], "role_id": role["id"]}); e != nil {
			return e
		}
		for _, menu := range []string{"Admins", "Roles", "Promotions", "Banners", "ContactUs", "customer", "Orders", "Transactions", "Brands", "Categories", "Points", "Gifts", "Users", "Settings"} {
			m, e := tx.Insert(ctx, "menus", domain.Record{"name": menu, "route": menu, "url": "/" + menu})
			if e != nil {
				return e
			}
			if _, e = tx.Insert(ctx, "role_menu", domain.Record{"role_id": role["id"], "menu_id": m["id"]}); e != nil {
				return e
			}
		}
		return nil
	})
	if e != nil {
		log.Fatal(e)
	}
	log.Print("Initial admin created")
}
