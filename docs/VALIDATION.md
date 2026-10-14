# Hasil verifikasi

Tanggal pembuatan: 13 Oktober 2026. Source Laravel: commit `0a6116c97a75da0e7cde7da83c9c75d81a30305b`. Runtime pengujian: Go 1.22.2 Linux/amd64 dengan CGO.

| Pemeriksaan | Hasil |
| --- | --- |
| Inventaris dan dispatch 181 kombinasi method/path | Lulus |
| `go test -race ./...`, 17 test | Lulus |
| `go vet ./...` | Lulus |
| Build api, migrate, seed, check-schema dengan `-buildvcs=false` | Lulus |
| `go mod verify` | Seluruh module terverifikasi |
| HTTP integration suite dengan SQLite | Lulus |
| HTTP integration suite dengan MariaDB 10.11.14 / driver MySQL GORM / race detector | Lulus |
| Migration database baru, preflight, initial admin seed | Lulus |
| Startup HTTP aktual, health, check-route, login dan me | Lulus |
| Syntax dan help comparison script Python | Lulus |

Pengujian mencakup envelope/pagination legacy, raw query, JWT guard terpisah, signature/expiry/algorithm, PHP bcrypt, legacy JWT opt-in, revocation logout, RBAC termasuk route lowercase, input sanitization, first-password setup dan token sekali pakai, registration rollback saat mail gagal, order items, ownership, member-level progress, upload multipart dan method override, cleanup file, query parameter binding, origin checks, serta login rate limit.

Pengujian financial concurrency memverifikasi bahwa kalkulasi ulang tidak menggandakan earning, dua claim yang melebihi saldo gabungan hanya mengizinkan satu pemotongan, approval gift bersamaan hanya memotong sekali, dan pemilihan alamat aktif tetap singular. Uji MariaDB memakai database sementara dan koneksi terpisah, bukan sekadar mock repository atau mutex aplikasi.

Batas verifikasi: Laravel tidak dijalankan untuk differential testing live; frontend, schema/data produksi, Google OAuth live, dan SMTP live tidak tersedia untuk diuji. Snapshot route dan bentuk response diturunkan dari source. Template Docker disertakan tetapi image belum dibangun. Perbedaan yang disengaja dan langkah pengujian staging ada di `MIGRATION.md`.
