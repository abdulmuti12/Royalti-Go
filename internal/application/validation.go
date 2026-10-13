package application

import (
	"context"
	"fmt"
	"net/mail"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/abdulmuti12/api-point-go/internal/domain"
)

// Validation mirrors Laravel field names. It runs before persistence and never
// accepts user-controlled columns outside a use case's allowlist.
func (s *Service) validate(ctx context.Context, in domain.Record, rules map[string]string, id string) error {
	errs := map[string][]string{}
	for field, spec := range rules {
		value, exists := in[field]
		empty := value == nil || domain.Text(value) == ""
		parts := strings.Split(spec, "|")
		if strings.Contains("|"+spec+"|", "|sometimes|") && !exists {
			continue
		}
		if empty {
			if strings.Contains("|"+spec+"|", "|required|") {
				errs[field] = []string{"The " + strings.ReplaceAll(field, "_", " ") + " field is required."}
			}
			continue
		}
		numeric := strings.Contains(spec, "integer") || strings.Contains(spec, "numeric")
		for _, rule := range parts {
			pair := strings.SplitN(rule, ":", 2)
			name := pair[0]
			arg := ""
			if len(pair) > 1 {
				arg = pair[1]
			}
			bad := false
			switch name {
			case "string":
				_, ok := value.(string)
				bad = !ok
			case "email":
				a, e := mail.ParseAddress(domain.Text(value))
				bad = e != nil || a.Address != domain.Text(value)
			case "integer", "numeric":
				_, e := strconv.ParseInt(domain.Text(value), 10, 64)
				bad = e != nil
			case "boolean":
				v := domain.Text(value)
				bad = v != "0" && v != "1" && v != "true" && v != "false"
			case "min", "max":
				limit, _ := strconv.ParseInt(arg, 10, 64)
				n := int64(utf8.RuneCountInString(domain.Text(value)))
				if numeric {
					n = domain.Integer(value)
				}
				if name == "min" {
					bad = n < limit
				} else {
					bad = n > limit
				}
			case "same":
				bad = domain.Text(value) != domain.Text(in[arg])
			case "confirmed":
				bad = domain.Text(value) != domain.Text(in[field+"_confirmation"])
			case "in":
				bad = true
				for _, option := range strings.Split(arg, ",") {
					if domain.Text(value) == option {
						bad = false
						break
					}
				}
			case "exists", "unique":
				args := strings.Split(arg, ",")
				column := "id"
				if len(args) > 1 {
					column = args[1]
				}
				q := domain.Q(domain.Equal(column, value))
				if name == "unique" && id != "" {
					q.Filters = append(q.Filters, domain.NotEqual("id", id))
				}
				n, e := s.Store.Count(ctx, args[0], q)
				if e != nil {
					return e
				}
				bad = (name == "exists" && n == 0) || (name == "unique" && n > 0)
			}
			if bad {
				errs[field] = append(errs[field], fmt.Sprintf("The %s field failed %s validation.", strings.ReplaceAll(field, "_", " "), name))
			}
		}
	}
	if len(errs) > 0 {
		return domain.Fail("Validasi gagal", errs)
	}
	return nil
}

func withoutFiles(rules map[string]string) map[string]string {
	o := map[string]string{}
	for k, v := range rules {
		if !strings.Contains(v, "|image") && !strings.Contains(v, "|file") {
			o[k] = v
		}
	}
	return o
}
