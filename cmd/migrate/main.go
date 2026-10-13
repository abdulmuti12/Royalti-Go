package main

import (
	"flag"
	"log"

	"github.com/abdulmuti12/api-point-go/internal/platform"
	"github.com/abdulmuti12/api-point-go/internal/repository/gormrepo"
)

func main() {
	business := flag.Bool("business", false, "create business tables in a NEW database only")
	flag.Parse()
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
	if *business {
		for _, t := range []string{"admins", "customers", "orders", "points", "gifts"} {
			if db.Migrator().HasTable(t) {
				log.Fatal("business migration refused: an existing business table was found; use the default security-only migration")
			}
		}
	}
	if e = gormrepo.Migrate(db, !*business); e != nil {
		log.Fatal(e)
	}
	log.Print("Migration completed")
}
