# API-Point Go + GORM

Port dari [API-Point](https://github.com/abdulmuti12/API-Point), berdasarkan commit `0a6116c97a75da0e7cde7da83c9c75d81a30305b` yang diambil pada 13 Oktober 2026. Acuan implementasi adalah `routes/api.php`, controller, service, resource, dan model aktual. Dokumen API lama di repository memuat beberapa modul yang sudah tidak terdaftar dan tidak dijadikan acuan route.

181 pasangan HTTP method/path dipertahankan, termasuk prefix `/api`, nama endpoint, singular/plural, PUT dan PATCH, request JSON/form/multipart, serta `_method` Laravel. Seluruh pasangan tersebut mempunyai handler Go. Sebanyak 25 pasangan di Laravel menunjuk method yang belum tersedia; versi Go melengkapinya. Detail ada di [ENDPOINTS.md](docs/ENDPOINTS.md) dan [MIGRATION.md](docs/MIGRATION.md).

## Yang tersedia

- JWT admin/customer dengan guard terpisah, RBAC berdasarkan menu, logout dengan revocation tersimpan di database, dan kompatibilitas hash bcrypt Laravel `$2y$`.
- Login/register customer, first-login password setup, verifikasi email, Google social authentication, SMTP STARTTLS/implicit TLS, dan reset password admin.
- Admin, role/menu, customer, category, brand, banner, promotion, logo, point rule, gift, member level, contact, order/items, dan transaction.
- Customer orders, transactions, addresses, address text, membership progress, gift request, dan claim history.
- Kalkulasi poin yang tidak menggandakan earning, pemotongan poin, approval/reject/complete gift, audit log, dan transaksi dengan row locking.
- Mapper respons legacy, termasuk perbedaan pagination antar modul, response role index tanpa envelope, dan respons email subscriber `201` dengan `status: true`.
- Migration eksplisit, pemeriksaan schema tanpa mengubah data, initial-admin seed, graceful shutdown, CORS/origin checks, upload validation, serta rate limiting per proses.

## Menjalankan dengan database Laravel yang ada

Gunakan Go yang masih didukung. Source kompatibel dengan Go 1.22+; pengujian di lingkungan pembuatan menggunakan Go 1.22.2. `vendor/` dan `go.sum` disertakan sehingga build tidak memerlukan pengunduhan dependency.

```bash
cp .env.example .env
```

Isi koneksi MySQL, `JWT_SECRET` minimal 32 byte, `APP_URL`, `FRONTEND_URL`, SMTP, dan Google client bila digunakan. Buat secret baru dengan `openssl rand -hex 32`. File `.env` Laravel bisa dijadikan referensi, tetapi gunakan `.env.example` Go untuk nama konfigurasi yang didukung.

```bash
go run -buildvcs=false ./cmd/check-schema
go run -buildvcs=false ./cmd/migrate
go run -buildvcs=false ./cmd/api
```

`cmd/migrate` tanpa flag hanya membuat tiga tabel state/security tambahan dan lock singleton. Business table tidak diubah saat startup maupun migration default. Jika `check-schema` melaporkan perbedaan, selesaikan pada salinan database staging terlebih dahulu. Khusus alamat, migration Laravel yang tersedia belum menjamin `addresses.user_id` bisa menyimpan UUID customer atau bahwa `recipient_name` tersedia.

Gunakan `STORAGE_PATH` yang menunjuk folder `storage/app/public` Laravel, atau salin isi folder tersebut. Pertahankan public base URL agar URL `/storage/...` dan pagination tetap cocok dengan frontend. Jika mengganti host, sesuaikan base URL API di FE atau arahkan reverse proxy lama ke Go.

## Database baru untuk pengembangan

Untuk SQLite, dibutuhkan compiler C ketika build karena driver SQLite menggunakan CGO. MySQL tetap menjadi target utama untuk transaksi produksi.

```dotenv
DB_CONNECTION=sqlite
DB_DATABASE=point.db
```

```bash
go run -buildvcs=false ./cmd/migrate -business
SEED_ADMIN_PASSWORD='ganti-dengan-password-kuat' go run -buildvcs=false ./cmd/seed -email admin@example.com
go run -buildvcs=false ./cmd/api
```

Flag `-business` ditolak apabila business table sudah ditemukan. Seed ditolak apabila admin sudah ada. Tidak ada akun atau password produksi bawaan. Data wilayah harus diisi menggunakan sumber `master_province`, `master_city`, `master_district`, `master_subdistrict`, dan `master_postal_code` yang sesuai; data wilayah tidak otomatis dikarang atau diimpor.

## Struktur dan pengujian

| Lokasi | Tanggung jawab |
| --- | --- |
| `cmd/` | Entry point server, migration, schema check, seed |
| `internal/domain/` | Value objects, error, port/interface; tanpa GORM/HTTP |
| `internal/application/` | Use case dan aturan bisnis per modul |
| `internal/repository/gormrepo/` | Model database, query parameterized, transaksi GORM |
| `internal/transport/httpapi/` | Route, request parsing, middleware, upload, respons legacy |
| `internal/platform/` | Konfigurasi, JWT, bcrypt, SMTP, Google integration |
| `docs/` | Kontrak route, perubahan perilaku, panduan migrasi |

```bash
go test -race ./...
go vet ./...
go build -buildvcs=false -o bin/api ./cmd/api
```

Untuk menguji row locking pada database MySQL/MariaDB yang terisolasi:

```bash
MYSQL_TEST_DSN='root:password@tcp(127.0.0.1:3306)/api_point_go_test?parseTime=true' go test -race ./internal/transport/httpapi -count=1
```

Test membuat dan menghapus database sementara dengan prefix `api_point_go_test_`. Nama database pada DSN wajib diawali `api_point_go_test`; gunakan akun database pengujian. Pengujian ini dijalankan pada MariaDB 10.11.14 melalui driver MySQL GORM.

Template Docker juga tersedia. Image Docker belum dibangun di lingkungan pembuatan. Build utama, unit/integration tests, race detector, module verification, dan vet diuji secara lokal; integrasi live SMTP, Google, FE, dan database produksi belum diuji.

```bash
docker build -t api-point-go .
docker run --rm --env-file .env -p 8000:8000 -v "$PWD/storage:/app/storage" api-point-go
```

Untuk membandingkan respons GET pada staging Laravel dan Go, gunakan `scripts/compare_api.py`. Petunjuk token dan batas perbandingan tersedia di [MIGRATION.md](docs/MIGRATION.md).
