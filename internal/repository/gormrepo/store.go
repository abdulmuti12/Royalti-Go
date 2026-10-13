package gormrepo

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/abdulmuti12/api-point-go/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Store struct{ db *gorm.DB }

func New(db *gorm.DB) *Store { return &Store{db: db} }

func (s *Store) query(ctx context.Context, table string, q domain.Query) (*gorm.DB, error) {
	schema, ok := schemas[table]
	if !ok {
		return nil, fmt.Errorf("unknown entity %q", table)
	}
	db := s.db.WithContext(ctx).Table(table)
	for _, f := range q.Filters {
		if !schema.Columns[f.Field] {
			return nil, fmt.Errorf("unknown column %q", f.Field)
		}
		col := clause.Column{Name: f.Field}
		switch f.Op {
		case "=":
			db = db.Where(clause.Eq{Column: col, Value: f.Value})
		case "!=":
			db = db.Where(clause.Neq{Column: col, Value: f.Value})
		case "IN":
			vals := []any{}
			v := reflect.ValueOf(f.Value)
			if v.IsValid() && (v.Kind() == reflect.Slice || v.Kind() == reflect.Array) {
				for i := 0; i < v.Len(); i++ {
					vals = append(vals, v.Index(i).Interface())
				}
			}
			db = db.Where(clause.IN{Column: col, Values: vals})
		case "LIKE":
			db = db.Where(clause.Like{Column: col, Value: f.Value})
		default:
			return nil, fmt.Errorf("unsupported filter operator")
		}
	}
	if q.Keyword != "" && len(q.AnyLike) > 0 {
		exprs := []clause.Expression{}
		for _, f := range q.AnyLike {
			if !schema.Columns[f] {
				return nil, fmt.Errorf("unknown search column")
			}
			exprs = append(exprs, clause.Like{Column: clause.Column{Name: f}, Value: "%" + q.Keyword + "%"})
		}
		db = db.Where(clause.Or(exprs...))
	}
	for _, o := range q.Sort {
		if !schema.Columns[o.Field] {
			return nil, fmt.Errorf("unknown sort column")
		}
		db = db.Order(clause.OrderByColumn{Column: clause.Column{Name: o.Field}, Desc: o.Desc})
	}
	if q.Limit > 0 {
		db = db.Limit(q.Limit)
	}
	if q.Offset > 0 {
		db = db.Offset(q.Offset)
	}
	if q.Lock {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	return db, nil
}

func normalize(r map[string]any) domain.Record {
	o := domain.Record{}
	for k, v := range r {
		switch x := v.(type) {
		case []byte:
			o[k] = string(x)
		case time.Time:
			o[k] = x.UTC().Format("2006-01-02T15:04:05.000000Z")
		default:
			o[k] = v
		}
	}
	return o
}
func (s *Store) List(ctx context.Context, t string, q domain.Query) ([]domain.Record, error) {
	db, e := s.query(ctx, t, q)
	if e != nil {
		return nil, e
	}
	rows := []map[string]any{}
	if e = db.Find(&rows).Error; e != nil {
		return nil, e
	}
	out := make([]domain.Record, 0, len(rows))
	for _, r := range rows {
		out = append(out, normalize(r))
	}
	return out, nil
}
func (s *Store) Find(ctx context.Context, t string, q domain.Query) (domain.Record, error) {
	q.Limit = 1
	rows, e := s.List(ctx, t, q)
	if e != nil {
		return nil, e
	}
	if len(rows) == 0 {
		return nil, domain.ErrNotFound
	}
	return rows[0], nil
}
func (s *Store) Count(ctx context.Context, t string, q domain.Query) (int64, error) {
	q.Sort = nil
	q.Limit = 0
	q.Offset = 0
	db, e := s.query(ctx, t, q)
	if e != nil {
		return 0, e
	}
	var n int64
	e = db.Count(&n).Error
	return n, e
}
func checkData(t string, r domain.Record) error {
	schema, ok := schemas[t]
	if !ok {
		return fmt.Errorf("unknown entity")
	}
	for k := range r {
		if !schema.Columns[k] {
			return fmt.Errorf("unknown write column %q", k)
		}
	}
	return nil
}
func (s *Store) Insert(ctx context.Context, t string, r domain.Record) (domain.Record, error) {
	if e := checkData(t, r); e != nil {
		return nil, e
	}
	data := domain.Copy(r)
	schema := schemas[t]
	if schema.UUID && domain.Text(data["id"]) == "" {
		id, e := domain.UUID()
		if e != nil {
			return nil, e
		}
		data["id"] = id
	}
	if schema.Timestamps {
		now := time.Now().UTC()
		if _, ok := data["created_at"]; !ok {
			data["created_at"] = now
		}
		data["updated_at"] = now
	}
	m := map[string]any(data)
	// Model metadata is essential: GORM then returns an auto-increment id for map inserts.
	if e := s.db.WithContext(ctx).Model(schema.Model).Create(&m).Error; e != nil {
		return nil, e
	}
	if _, ok := data["id"]; ok {
		return s.Find(ctx, t, domain.Q(domain.Equal("id", data["id"])))
	}
	return normalize(m), nil
}
func (s *Store) Update(ctx context.Context, t string, q domain.Query, r domain.Record) error {
	if len(q.Filters) == 0 {
		return fmt.Errorf("refusing unscoped update")
	}
	if e := checkData(t, r); e != nil {
		return e
	}
	data := domain.Copy(r)
	if schemas[t].Timestamps {
		data["updated_at"] = time.Now().UTC()
	}
	db, e := s.query(ctx, t, q)
	if e != nil {
		return e
	}
	return db.Updates(map[string]any(data)).Error
}
func (s *Store) Delete(ctx context.Context, t string, q domain.Query) error {
	if len(q.Filters) == 0 {
		return fmt.Errorf("refusing unscoped delete")
	}
	db, e := s.query(ctx, t, q)
	if e != nil {
		return e
	}
	return db.Delete(map[string]any{}).Error
}
func (s *Store) Transaction(ctx context.Context, fn func(domain.Store) error) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { return fn(New(tx)) })
}

// Migrate creates a new database only. Existing Laravel databases must use only
// Migrate(db, true): AutoMigrate must never rewrite production business tables.
func Migrate(db *gorm.DB, securityOnly bool) error {
	names := []string{}
	for name := range schemas {
		if !securityOnly || name == "point_go_tokens" || name == "point_go_revoked_tokens" || name == "point_go_locks" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		if e := db.AutoMigrate(schemas[name].Model); e != nil {
			return e
		}
	}
	for _, key := range []string{"points", "promotions", "banner", "logos"} {
		e := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&Lock{Key: key}).Error
		if e != nil {
			return e
		}
	}
	return nil
}
func IsNotFound(e error) bool { return errors.Is(e, domain.ErrNotFound) }
