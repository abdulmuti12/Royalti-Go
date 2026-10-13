package httpapi

import (
	"html"
	"regexp"
	"strings"

	"github.com/abdulmuti12/api-point-go/internal/domain"
)

var scriptTag = regexp.MustCompile(`(?is)<\s*script\b[^>]*>.*?<\s*/\s*script\s*>`)
var dangerousTag = regexp.MustCompile(`(?i)<\s*(?:script|iframe|object|embed|form|input|link|meta|base|applet|video|audio|source|background|body|div|span|marquee|layer|bleg)\b[^>]*>`)
var eventQuoted = regexp.MustCompile(`(?i)on\w+\s*=\s*["'][^"']*["']`)
var eventUnquoted = regexp.MustCompile(`(?i)on\w+\s*=\s*[^\s>]*`)

// Keep the request sanitation used by the PHP routes. This matters even for
// bcrypt: a password hashed after PHP entity-escaping must be checked against
// the same escaped input during login in the Go port.
func sanitizeLegacy(value any) any {
	switch v := value.(type) {
	case string:
		v = html.UnescapeString(strings.ReplaceAll(v, "\x00", ""))
		v = scriptTag.ReplaceAllString(v, "")
		v = dangerousTag.ReplaceAllString(v, "")
		v = eventQuoted.ReplaceAllString(v, "")
		v = eventUnquoted.ReplaceAllString(v, "")
		v = html.EscapeString(v)
		v = strings.ReplaceAll(strings.ReplaceAll(v, "&#39;", "&apos;"), "&#34;", "&quot;")
		return strings.TrimSpace(v)
	case domain.Record:
		for k, x := range v {
			v[k] = sanitizeLegacy(x)
		}
	case map[string]any:
		for k, x := range v {
			v[k] = sanitizeLegacy(x)
		}
	case []any:
		for i, x := range v {
			v[i] = sanitizeLegacy(x)
		}
	}
	return value
}
func hasLegacySanitizer(route Route) bool {
	if strings.HasPrefix(route.Path, "/api/admins/") {
		return true
	}
	if !strings.HasPrefix(route.Path, "/api/customers/") {
		return false
	}
	for _, prefix := range []string{"/api/customers/customer-data", "/api/customers/transactions"} {
		if strings.HasPrefix(route.Path, prefix) {
			return false
		}
	}
	return route.Action != "logout" && route.Action != "change-password" && route.Action != "claim-gift"
}
