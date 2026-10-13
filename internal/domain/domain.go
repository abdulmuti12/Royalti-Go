// Package domain defines the ports and value objects shared by application use cases.
// It has no dependency on HTTP, GORM, or a database driver.
package domain

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Record map[string]any

func Text(v any) string {
	if v == nil {
		return ""
	}
	if b, ok := v.([]byte); ok {
		return string(b)
	}
	return fmt.Sprint(v)
}

func Integer(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case uint64:
		return int64(n)
	case float64:
		return int64(n)
	case json.Number:
		i, _ := n.Int64()
		return i
	}
	i, _ := strconv.ParseInt(Text(v), 10, 64)
	return i
}

func Bool(v any) bool {
	s := strings.ToLower(Text(v))
	return s == "true" || s == "1"
}

func Pick(r Record, fields ...string) Record {
	o := Record{}
	for _, f := range fields {
		o[f] = r[f]
	}
	return o
}

func Copy(r Record) Record {
	o := Record{}
	for k, v := range r {
		o[k] = v
	}
	return o
}

func RandomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
func UUID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	// UUIDv7 keeps time in the leading bytes, matching the ordered IDs used by
	// Laravel's UUID models and making id-desc lists chronological across time.
	ts := uint64(time.Now().UnixMilli())
	for i := 5; i >= 0; i-- {
		b[i] = byte(ts)
		ts >>= 8
	}
	b[6] = (b[6] & 0x0f) | 0x70
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}

var ErrNotFound = errors.New("record not found")

// Failure preserves legacy validation/business errors without exposing SQL errors.
type Failure struct {
	Label   string
	Message any
	Status  int
	Body    Record
}

func (e *Failure) Error() string { return e.Label }
func Fail(label string, message any) error {
	return &Failure{Label: label, Message: message, Status: 401}
}
func Reject(status int, body Record) error { return &Failure{Status: status, Body: body} }

type Filter struct {
	Field, Op string
	Value     any
}
type Sort struct {
	Field string
	Desc  bool
}
type Query struct {
	Filters       []Filter
	AnyLike       []string
	Keyword       string
	Sort          []Sort
	Limit, Offset int
	Lock          bool
}

func Equal(field string, value any) Filter    { return Filter{field, "=", value} }
func NotEqual(field string, value any) Filter { return Filter{field, "!=", value} }
func In(field string, value any) Filter       { return Filter{field, "IN", value} }
func Q(filters ...Filter) Query               { return Query{Filters: filters} }

// Store is a transaction-aware persistence port. Callbacks receive only the
// transaction-scoped store, so use cases cannot accidentally escape a transaction.
type Store interface {
	List(context.Context, string, Query) ([]Record, error)
	Find(context.Context, string, Query) (Record, error)
	Count(context.Context, string, Query) (int64, error)
	Insert(context.Context, string, Record) (Record, error)
	Update(context.Context, string, Query, Record) error
	Delete(context.Context, string, Query) error
	Transaction(context.Context, func(Store) error) error
}

type Principal struct {
	ID, Kind, Name, JTI string
	Expires             time.Time
}
type Tokens interface {
	Issue(Principal) (string, error)
	Parse(string, string) (Principal, error)
}
type Passwords interface {
	Hash(string) (string, error)
	Check(string, string) bool
}
type Mailer interface {
	Send(context.Context, string, string, string) error
}
type GoogleIdentity struct{ Subject, Email, Name, Avatar string }
type IdentityProvider interface {
	Verify(context.Context, string, string) (GoogleIdentity, error)
}

type Command struct {
	Action, Table, ID string
	Input             Record
	Principal         Principal
}
type Result struct {
	Data    any
	Message string
	Body    Record
	Status  int
}

func Success(data any, message string) Result {
	return Result{Data: data, Message: message, Status: 200}
}

type Page struct {
	Items        []Record
	Total        int64
	Number, Size int
	Style        string
}
