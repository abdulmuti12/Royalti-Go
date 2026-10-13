package httpapi

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/abdulmuti12/api-point-go/internal/domain"
)

var itemKey = regexp.MustCompile(`^items\[(\d+)\]\[([a-z_]+)\]$`)
var menuKey = regexp.MustCompile(`^menus\[(\d*)\]$`)

func decodeInput(w http.ResponseWriter, r *http.Request) (domain.Record, error) {
	in := domain.Record{}
	for key, values := range r.URL.Query() {
		if len(values) > 0 {
			in[key] = strings.TrimSpace(values[0])
		}
	}
	if r.Body == nil || r.ContentLength == 0 {
		return in, nil
	}
	r.Body = http.MaxBytesReader(w, r.Body, 100<<20)
	contentType, _, e := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if e != nil && r.Header.Get("Content-Type") != "" {
		return nil, fmt.Errorf("invalid Content-Type")
	}
	if contentType == "application/json" {
		decoder := json.NewDecoder(r.Body)
		decoder.UseNumber()
		body := domain.Record{}
		if e := decoder.Decode(&body); e != nil {
			if e == io.EOF {
				return in, nil
			}
			return nil, e
		}
		var trailing any
		if e := decoder.Decode(&trailing); e != io.EOF {
			return nil, fmt.Errorf("body must contain one JSON object")
		}
		for k, v := range body {
			in[k] = v
		}
		return trimInput(in).(domain.Record), nil
	}
	if contentType == "multipart/form-data" {
		if e := r.ParseMultipartForm(8 << 20); e != nil {
			return nil, e
		}
	} else {
		if e := r.ParseForm(); e != nil {
			return nil, e
		}
	}
	items := map[int]map[string]any{}
	menus := map[int]any{}
	nextMenu := 0
	for key, values := range r.PostForm {
		if len(values) == 0 {
			continue
		}
		if m := itemKey.FindStringSubmatch(key); m != nil {
			i, e := strconv.Atoi(m[1])
			if e != nil || i > 1000 {
				return nil, fmt.Errorf("too many items")
			}
			if items[i] == nil {
				items[i] = map[string]any{}
			}
			items[i][m[2]] = strings.TrimSpace(values[0])
			continue
		}
		if m := menuKey.FindStringSubmatch(key); m != nil {
			if m[1] != "" {
				i, e := strconv.Atoi(m[1])
				if e != nil || i > 1000 {
					return nil, fmt.Errorf("too many menus")
				}
				menus[i] = values[0]
			} else {
				for _, v := range values {
					menus[nextMenu] = v
					nextMenu++
				}
			}
			continue
		}
		value := strings.TrimSpace(values[0])
		if key == "items" || key == "menus" {
			var arr []any
			d := json.NewDecoder(strings.NewReader(value))
			d.UseNumber()
			if e := d.Decode(&arr); e == nil {
				in[key] = arr
				continue
			}
		}
		if value == "" {
			in[key] = nil
		} else {
			in[key] = value
		}
	}
	if len(items) > 0 {
		indices := []int{}
		for i := range items {
			indices = append(indices, i)
		}
		sort.Ints(indices)
		arr := []any{}
		for _, i := range indices {
			arr = append(arr, items[i])
		}
		in["items"] = arr
	}
	if len(menus) > 0 {
		indices := []int{}
		for i := range menus {
			indices = append(indices, i)
		}
		sort.Ints(indices)
		arr := []any{}
		for _, i := range indices {
			arr = append(arr, menus[i])
		}
		in["menus"] = arr
	}
	return in, nil
}
func trimInput(value any) any {
	switch v := value.(type) {
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return nil
		}
		return s
	case domain.Record:
		for k, x := range v {
			if k == "password" || k == "password_confirmation" || k == "current_password" {
				continue
			}
			v[k] = trimInput(x)
		}
	case map[string]any:
		for k, x := range v {
			if k == "password" || k == "password_confirmation" || k == "current_password" {
				continue
			}
			v[k] = trimInput(x)
		}
	case []any:
		for i, x := range v {
			v[i] = trimInput(x)
		}
	}
	return value
}
