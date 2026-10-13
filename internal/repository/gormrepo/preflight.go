package gormrepo

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// CheckSchema is read-only. A dump of the actual deployed Laravel schema,
// rather than the old API_DOCUMENTATION.md, is authoritative for migration.
func CheckSchema(db *gorm.DB) ([]string, error) {
	issues := []string{}
	for name, schema := range schemas {
		if strings.HasPrefix(name, "point_go_") {
			continue
		}
		if !db.Migrator().HasTable(name) {
			issues = append(issues, "missing table: "+name)
			continue
		}
		columns, e := db.Migrator().ColumnTypes(schema.Model)
		if e != nil {
			return nil, e
		}
		present := map[string]string{}
		for _, c := range columns {
			present[c.Name()] = strings.ToLower(c.DatabaseTypeName())
		}
		for column := range schema.Columns {
			if column == "remember_token" || column == "province" || column == "city" || column == "district" || column == "address_detail" {
				continue
			}
			if _, ok := present[column]; !ok {
				issues = append(issues, fmt.Sprintf("missing column: %s.%s", name, column))
			}
		}
		if name == "customers" || name == "addresses" {
			key := "id"
			if name == "addresses" {
				key = "user_id"
			}
			typ := present[key]
			if typ != "" && !strings.Contains(typ, "char") && !strings.Contains(typ, "text") {
				issues = append(issues, fmt.Sprintf("%s.%s must store customer UUIDs (currently %s)", name, key, typ))
			}
		}
	}
	return issues, nil
}
