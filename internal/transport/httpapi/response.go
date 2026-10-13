package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/abdulmuti12/api-point-go/internal/domain"
)

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if e := json.NewEncoder(w).Encode(body); e != nil {
		slog.Error("response encoding failed", "error", e)
	}
}
func legacyRaw(r *http.Request, route Route) bool {
	value := r.URL.Query().Get("raw")
	if route.Table == "admins" {
		b, _ := strconv.ParseBool(value)
		return b
	}
	return value != "" && value != "0"
}

func paginate(p domain.Page, base string, raw bool) any {
	if raw {
		return p.Items
	}
	last := int((p.Total + int64(p.Size) - 1) / int64(p.Size))
	if last < 1 {
		last = 1
	}
	url := func(n int) any {
		if n < 1 || n > last {
			return nil
		}
		return base + "?page=" + strconv.Itoa(n)
	}
	links := domain.Record{"first": url(1), "last": url(last), "prev": url(p.Number - 1), "next": url(p.Number + 1)}
	var from, to any
	if len(p.Items) > 0 {
		from = (p.Number-1)*p.Size + 1
		to = (p.Number-1)*p.Size + len(p.Items)
	}
	nav := []domain.Record{{"url": links["prev"], "label": "&laquo; Previous", "active": false}}
	// Laravel uses a sliding pagination window, rather than emitting every page.
	pages := []int{}
	if last < 14 {
		for n := 1; n <= last; n++ {
			pages = append(pages, n)
		}
	} else if p.Number <= 7 {
		for n := 1; n <= 10; n++ {
			pages = append(pages, n)
		}
		pages = append(pages, 0, last-1, last)
	} else if p.Number > last-7 {
		pages = append(pages, 1, 2, 0)
		for n := last - 9; n <= last; n++ {
			pages = append(pages, n)
		}
	} else {
		pages = append(pages, 1, 2)
		start, end := p.Number-3, p.Number+3
		if start < 3 {
			start = 3
		}
		if end > last-2 {
			end = last - 2
		}
		pages = append(pages, 0)
		for n := start; n <= end; n++ {
			pages = append(pages, n)
		}
		pages = append(pages, 0)
		pages = append(pages, last-1, last)
	}
	for _, n := range pages {
		if n == 0 {
			nav = append(nav, domain.Record{"url": nil, "label": "...", "active": false})
		} else {
			nav = append(nav, domain.Record{"url": url(n), "label": fmt.Sprint(n), "active": n == p.Number})
		}
	}
	nav = append(nav, domain.Record{"url": links["next"], "label": "Next &raquo;", "active": false})
	meta := domain.Record{"current_page": p.Number, "from": from, "last_page": last, "links": nav, "path": base, "per_page": p.Size, "to": to, "total": p.Total}
	switch p.Style {
	case "flat":
		out := domain.Copy(meta)
		out["data"] = p.Items
		out["first_page_url"] = links["first"]
		out["last_page_url"] = links["last"]
		out["prev_page_url"] = links["prev"]
		out["next_page_url"] = links["next"]
		return out
	case "paginator":
		out := domain.Copy(meta)
		out["data"] = p.Items
		out["first_page_url"] = links["first"]
		out["last_page_url"] = links["last"]
		out["prev_page_url"] = links["prev"]
		out["next_page_url"] = links["next"]
		return out
	case "order":
		return domain.Record{"data": p.Items, "meta": domain.Record{"current_page": p.Number, "last_page": last, "per_page": p.Size, "total": p.Total}, "links": domain.Record{"first_page_url": links["first"], "last_page_url": links["last"], "prev_page_url": links["prev"], "next_page_url": links["next"]}}
	case "plain":
		return domain.Record{"data": p.Items, "meta": meta, "links": links}
	default:
		return domain.Record{"data": domain.Record{"data": p.Items}, "meta": meta, "links": links}
	}
}

func failure(w http.ResponseWriter, route Route, e error) {
	var f *domain.Failure
	if errors.As(e, &f) {
		if f.Body != nil {
			writeJSON(w, f.Status, f.Body)
			return
		}
		status := f.Status
		if status == 0 {
			status = 401
		}
		if f.Label == "Validasi gagal" {
			if route.Table == "admins" && (route.Action == "create" || route.Action == "update") {
				f.Label = "errors"
			}
			if route.Table == "brands" && route.Action == "create" || route.Table == "logos" && (route.Action == "create" || route.Action == "update") {
				f.Label = "Gagal"
			}
			if route.Table == "roles" && route.Action == "create" {
				f.Label = "Validation Error"
			}
		}
		if route.Action == "set-password" || route.Action == "verify-email" {
			writeJSON(w, 400, domain.Record{"success": false, "message": f.Message})
			return
		}
		if route.Action == "social-login" || route.Action == "social-register" {
			body := domain.Record{"success": false, "message": f.Message, "status": 401}
			if route.Action == "social-register" {
				body["errors"] = nil
				body["duplicate_fields"] = nil
			}
			writeJSON(w, 401, body)
			return
		}
		if f.Label == "Validasi gagal" && (route.Action == "change-password" || route.Action == "reset-password" || route.Action == "resend-verification") {
			writeJSON(w, 422, domain.Record{"message": "The given data was invalid.", "errors": f.Message})
			return
		}
		if route.Action == "register" {
			writeJSON(w, 422, domain.Record{"success": false, "message": "Validation Error", "errors": f.Message, "duplicate_fields": nil, "status": 422})
			return
		}
		if (route.Action == "order-detail" || route.Action == "customer-order-detail") && f.Label == "Not Found" {
			writeJSON(w, 404, domain.Record{"success": false, "message": f.Message, "status": 404})
			return
		}
		writeJSON(w, status, domain.Record{"success": false, "message": f.Message, "error": f.Label, "status": status})
		return
	}
	if errors.Is(e, domain.ErrNotFound) {
		writeJSON(w, 401, domain.Record{"success": false, "message": "Data tidak ditemukan", "error": "Data tidak ditemukan", "status": 401})
		return
	}
	slog.Error("request failed", "method", route.Method, "path", route.Path, "error", e)
	writeJSON(w, 500, domain.Record{"success": false, "message": "Terjadi kesalahan pada server", "status": 500})
}

func respond(w http.ResponseWriter, r *http.Request, route Route, result domain.Result, appURL string) {
	status := result.Status
	if status == 0 {
		status = 200
	}
	if result.Body != nil {
		writeJSON(w, status, result.Body)
		return
	}
	data := result.Data
	if p, ok := data.(domain.Page); ok {
		data = paginate(p, strings.TrimRight(appURL, "/")+r.URL.Path, legacyRaw(r, route))
		if route.Table == "roles" && route.Action == "index" {
			writeJSON(w, status, data)
			return
		}
	}
	writeJSON(w, status, domain.Record{"success": true, "message": result.Message, "data": data, "status": 200})
}
