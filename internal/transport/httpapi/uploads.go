package httpapi

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/abdulmuti12/api-point-go/internal/domain"
)

func fileFields(table string) []string {
	switch table {
	case "gifts":
		return []string{"image", "image2", "image3"}
	case "brands", "category", "logos":
		return []string{"image"}
	case "member_levels":
		return []string{"file"}
	case "promotions":
		return []string{"file", "file2", "file3", "file4", "file5"}
	case "banner":
		return []string{"file", "file2", "file3", "file4"}
	}
	return nil
}

// ensureNoSymlink rejects a destination whose parent chain under the storage
// root contains a symlink. An attacker who can plant a symlink inside the
// storage tree would otherwise redirect O_CREATE writes to a location outside
// the root.
func (a *API) ensureNoSymlink(destination string) error {
	root := filepath.Clean(a.Config.Storage)
	dir := filepath.Dir(destination)
	for d := dir; d != "" && d != "/"; d = filepath.Dir(d) {
		if d == root {
			break
		}
		info, e := os.Lstat(d)
		if e != nil {
			if os.IsNotExist(e) {
				continue
			}
			return e
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return domain.Fail("Upload failed", "Storage path contains a symlink")
		}
	}
	if info, e := os.Lstat(destination); e == nil && info.Mode()&os.ModeSymlink != 0 {
		return domain.Fail("Upload failed", "Storage path is a symlink")
	}
	return nil
}
func (a *API) uploads(r *http.Request, route Route, in domain.Record) ([]string, error) {
	saved := []string{}
	fields := fileFields(route.Table)
	if route.Action != "create" && route.Action != "update" {
		return saved, nil
	}
	for _, field := range fields {
		delete(in, field) // A client cannot inject a storage path instead of an uploaded file.
		if r.MultipartForm == nil {
			continue
		}
		files := r.MultipartForm.File[field]
		if len(files) == 0 {
			continue
		}
		if len(files) != 1 {
			return saved, domain.Fail("Upload failed", "Satu file per field diperlukan")
		}
		f := files[0]
		maxSize := int64(5 << 20)
		if route.Table == "category" {
			maxSize = 15 << 20
			if route.Action == "create" {
				maxSize = 88408 * 1024
			}
		}
		if route.Table == "banner" {
			maxSize = 20 << 20
		}
		if f.Size > maxSize {
			return saved, domain.Fail("Upload failed", "Ukuran file melebihi batas upload")
		}
		ext := strings.ToLower(filepath.Ext(f.Filename))
		allowed := map[string]string{".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".png": "image/png", ".gif": "image/gif", ".webp": "image/webp"}
		if route.Table != "member_levels" {
			delete(allowed, ".webp")
		}
		if route.Table == "promotions" || route.Table == "banner" {
			delete(allowed, ".gif")
		}
		if route.Table == "banner" {
			allowed[".mp4"] = "video/mp4"
			allowed[".mov"] = "video/quicktime"
			allowed[".avi"] = "video/avi"
			allowed[".webm"] = "video/webm"
		}
		expected, ok := allowed[ext]
		if !ok {
			return saved, domain.Fail("Upload failed", "File extension not allowed")
		}
		reader, e := f.Open()
		if e != nil {
			return saved, e
		}
		head := make([]byte, 512)
		n, e := reader.Read(head)
		if e != nil && e != io.EOF {
			reader.Close()
			return saved, e
		}
		head = head[:n]
		actual := http.DetectContentType(head)
		if (ext == ".mp4" || ext == ".mov") && len(head) >= 12 && string(head[4:8]) == "ftyp" {
			actual = expected
		}
		if ext == ".avi" && len(head) >= 12 && string(head[:4]) == "RIFF" && string(head[8:12]) == "AVI " {
			actual = expected
		}
		if actual != expected {
			reader.Close()
			return saved, domain.Fail("Upload failed", "File signature verification failed")
		}
		if _, e = reader.Seek(0, 0); e != nil {
			reader.Close()
			return saved, e
		}
		directory := map[string]string{"category": "categories", "banner": "banners"}[route.Table]
		if directory == "" {
			directory = route.Table
		}
		token, e := domain.RandomToken()
		if e != nil {
			reader.Close()
			return saved, e
		}
		relative := directory + "/file_" + token + ext
		destination := filepath.Join(a.Config.Storage, filepath.FromSlash(relative))
		if e = a.ensureNoSymlink(destination); e != nil {
			reader.Close()
			return saved, e
		}
		if e = os.MkdirAll(filepath.Dir(destination), 0750); e != nil {
			reader.Close()
			return saved, e
		}
		var out *os.File
		if runtime.GOOS == "windows" {
			out, e = os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0640)
		} else {
			out, e = os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY|syscall.O_NOFOLLOW, 0640)
		}
		if e != nil {
			reader.Close()
			return saved, e
		}
		_, copyErr := io.Copy(out, io.LimitReader(reader, maxSize+1))
		closeErr := out.Close()
		reader.Close()
		if copyErr != nil || closeErr != nil {
			os.Remove(destination)
			if copyErr != nil {
				return saved, copyErr
			}
			return saved, closeErr
		}
		saved = append(saved, relative)
		in[field] = relative
	}
	if route.Table == "brands" && route.Action == "create" && in["image"] == nil {
		return saved, domain.Fail("Gagal", domain.Record{"image": []string{"The image field is required."}})
	}
	return saved, nil
}
func (a *API) removeFiles(paths []string) {
	for _, p := range paths {
		if p == "" {
			continue
		}
		clean := filepath.Clean(filepath.FromSlash(p))
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			continue
		}
		if e := os.Remove(filepath.Join(a.Config.Storage, clean)); e != nil && !os.IsNotExist(e) {
			a.Logger.Warn("file cleanup failed", "error", e)
		}
	}
}
func (a *API) retiredFiles(ctxReq *http.Request, route Route, in domain.Record) ([]string, error) {
	if route.Action != "update" && route.Action != "delete" {
		return nil, nil
	}
	fields := fileFields(route.Table)
	if len(fields) == 0 {
		return nil, nil
	}
	row, e := a.Service.Find(ctxReq.Context(), route.Table, ctxReq.PathValue("id"))
	if e != nil {
		return nil, e
	}
	out := []string{}
	for _, f := range fields {
		if route.Action == "delete" || in[f] != nil {
			if old := domain.Text(row[f]); old != "" {
				out = append(out, old)
			}
		}
	}
	return out, nil
}
