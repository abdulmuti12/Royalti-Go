# Struktur clean code

Alur dependency: HTTP transport memanggil application use case; application hanya bergantung pada port dan value object domain. GORM repository dan adapter platform mengimplementasikan port tersebut. `cmd/api` merangkai dependency. Tidak ada query GORM di HTTP handler atau domain.

`domain.Store` menyediakan query dengan field/operator terstruktur dan callback transaksi yang menerima store milik transaksi itu sendiri. Implementasi repository memeriksa nama tabel/kolom pada allowlist, mengikat nilai query sebagai parameter, menolak update/delete tanpa filter, serta mengembalikan `ErrNotFound` yang konsisten. Model GORM memakai nama kolom/tabel aktual Laravel.

`application` dipisahkan menjadi auth, catalog, public data, addresses, orders, transactions, dan loyalty. CRUD yang mempunyai pola sama berbagi validasi/spec. Operasi finansial mempunyai use case khusus; tidak dijalankan sebagai CRUD generik. Row locking dan pengecekan saldo/status dilakukan dalam transaksi yang sama dengan perubahan saldo dan history/audit.

`httpapi` merupakan adapter kompatibilitas. Variasi envelope legacy, pagination, raw query, typo field, multipart/form parsing, method override, CORS, rate limiting, serta upload lifecycle berada di batas API. Value object `Record` digunakan untuk data legacy yang bentuknya berbeda-beda, sementara model persistence tetap bertipe. Ini pilihan pragmatis untuk migrasi kontrak lama; bukan klaim bahwa seluruh domain sudah dimodelkan dengan DTO bertipe terpisah.

`platform` menangani hash password, signing/parsing JWT, Google verification, SMTP, dan konfigurasi. Integrasi tersebut bisa diganti melalui interface domain pada test. SMTP dan Google tidak memakai response dari client sebagai identitas yang dipercaya begitu saja.

Schema production tidak auto-migrate. Tabel baru untuk token/lock dibuat eksplisit. Persistence model migration untuk database baru tidak mencoba menyalin seluruh sejarah migration PHP atau data legacy.
