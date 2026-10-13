package main

import (
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/abdulmuti12/api-point-go/internal/platform"
	"github.com/abdulmuti12/api-point-go/internal/repository/gormrepo"
)

func main() {
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
	issues, e := gormrepo.CheckSchema(db)
	if e != nil {
		log.Fatal(e)
	}
	sort.Strings(issues)
	for _, issue := range issues {
		fmt.Println(issue)
	}
	if len(issues) > 0 {
		os.Exit(1)
	}
	fmt.Println("Business schema is compatible with the Go persistence models")
}
