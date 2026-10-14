# Kontrak dan migrasi frontend

Source Laravel yang diaudit: commit `0a6116c97a75da0e7cde7da83c9c75d81a30305b`. Inventaris lengkap terdapat di `route-contract.json` dan `ENDPOINTS.md`. Inventaris berisi 181 kombinasi method/path, dengan 156 kombinasi yang menunjuk method controller tersedia dan 25 yang menunjuk method belum tersedia. Keberadaan method belum membuktikan bahwa seluruh konfigurasi deployment Laravel berfungsi; contoh customer profile merujuk mapper yang tidak ada di source.

## Kontrak yang dipertahankan

| Modul | Bentuk response |
| --- | --- |
| Success umum | `{success: true, message, data, status: 200}` |
| Error `BaseController::sendError` | HTTP 401, `{success: false, message, error, status: 401}`; argument status tambahan pada PHP aslinya tidak dipakai |
| Admin/customer index | Envelope umum; `data` berupa paginator datar |
| Category/brand/banner/promotion/point/logo/contact/transaction index | Envelope umum; `data.data.data` berisi array dan `data.meta`/`data.links` berada di sampingnya |
| Gift/member level index | Envelope umum; `data.data` berisi array, dengan `meta`/`links` |
| Role index | `{data: {data: [...]}, meta, links}` tanpa envelope umum |
| Admin get-orders | Envelope umum; `data.data` array, meta ringkas, links `*_page_url` |
| Customer get-orders | Envelope umum; `data` langsung berupa array |
| Subscriber success | HTTP 201, `{status: true, message: "Email Sudah Ada", data}` |
| Customer login belum aktif | HTTP 403 dengan `needs_password_setup`, `setup_token`, dan `email` |
| Set-password success | `{success, message, data}` tanpa `status` |

`raw` mempertahankan truthiness PHP untuk sebagian besar resource, sehingga `raw=false` sebagai string menghasilkan array. Admin index mengikuti `FILTER_VALIDATE_BOOLEAN`, sehingga `raw=false` tetap menghasilkan paginator. Pagination menyertakan URL, total, from/to, dan window navigasi.

Sanitasi input string pada route yang memakai `SanitizeXssInput` Laravel dipertahankan, termasuk entity encoding sebelum pemeriksaan password pada login. RBAC membandingkan route menu tanpa membedakan huruf besar/kecil seperti collation MySQL source.

Field typo `decription` pada BannerResource tetap dipertahankan. Admin order views juga mempertahankan cast integer pada `customer.id` dari source Laravel, meskipun customer sebenarnya menggunakan UUID. UUID yang benar tersedia melalui endpoint customer. Mengubah field tersebut harus dilakukan bersama penyesuaian kontrak FE.

Nama business table tetap mengikuti model Laravel: antara lain `category`, `banner`, `transaction`, `address_text`, `orders`, `order_items`, `points`, dan `redeem_points`. Tidak ada pemaksaan nama plural GORM. Money dan point dibaca sebagai integer; request monetary pecahan ditolak agar nilai tidak terpotong diam-diam. Kolom nullable dipetakan ke null dan tanggal ke format ISO yang dapat dibaca frontend.

## Perubahan perilaku yang disengaja

- 25 kombinasi route yang belum memiliki method PHP diimplementasikan dengan use case yang sesuai. Contohnya CRUD contact, customer update/delete, admin transaction CRUD, admin order detail/update/delete, dan alias customer orders. Bentuk respons pada route ini tidak bisa dianggap salinan respons sukses Laravel karena respons sukses aslinya belum tersedia.
- Ownership customer transaction diperiksa melalui `orders.customer_id`, bukan perbandingan keliru antara integer `order_id` dan UUID customer. Customer tidak dapat mengubah/menghapus transaksi sukses atau mengonfirmasi status pembayaran `success`; konfirmasi ada di admin transaction API. Request sukses yang semula dapat dilakukan customer sekarang ditolak.
- `customer-data` tetap berada di guard admin sesuai `jwt.auth` pada route source, dengan izin menu customer benar-benar ditegakkan. Hal ini menutup akses yang sebelumnya hanya memanggil pengecekan izin tetapi mengabaikan hasilnya.
- Perhitungan, klaim poin, dan perubahan status gift claim dijalankan di transaksi database. Row customer dikunci sebelum balance/claim dibaca ulang. Lock singleton mengatur perubahan active point rule, promotion, banner, dan logo. Approval berulang tidak memotong poin dua kali.
- Pemilihan redeem terbaru menggunakan waktu pembuatan dan ID, sehingga tidak bergantung hanya pada urutan UUID. ID baru menggunakan UUIDv7.
- Daftar gift mengambil claim terbaru dan memeriksa pending claim yang masih ada. Source menggunakan `keyBy` setelah urutan descending sehingga claim lama dapat menimpa claim terbaru.
- Upload menolak path string pengganti file dan SVG aktif. Tipe gambar mengikuti field; batas umum upload service adalah 5 MB, category mengikuti 88408 KB saat create/15 MB saat update. Banner mendukung video sesuai deklarasi controller dengan batas 20 MB; upload service PHP sebelumnya justru menolak video.
- File lama dihapus setelah perubahan database berhasil. File upload baru dibersihkan jika validasi atau transaksi gagal. Update deskripsi logo disimpan; source hanya memvalidasinya tanpa menyimpan nilai baru.
- Error validasi menggunakan nama field yang sama; beberapa kalimat error/rincian text berbeda. Error internal tidak menampilkan SQL atau stack trace. SMTP failure membatalkan pembuatan akun, alih-alih menyatakan email telah dikirim. Pengiriman SMTP dan commit database bukan distributed transaction: penerimaan email oleh SMTP tidak menjamin delivery ke inbox dan commit masih bisa gagal setelah email diterima SMTP.
- Admin/customer JWT tidak dapat dipakai silang. Rate limiting berlaku per route dan IP koneksi langsung, per proses. Origin/Referer diperiksa untuk state-changing requests; request tanpa kedua header tetap diterima sebagaimana source Laravel. Untuk banyak instance di belakang proxy, terapkan shared rate limiting/IP forwarding pada reverse proxy.

## Schema dan sesi saat cutover

1. Siapkan salinan schema/data produksi dan storage di staging. Source migration Laravel lama mempunyai perubahan schema yang tidak konsisten dengan sebagian model. Jangan menjalankan migration lama yang menghapus/recreate order hanya untuk mencoba Go.
2. Jalankan `cmd/check-schema`. Perbaiki schema atau mapping legacy pada staging berdasarkan data aktual. Laporan `addresses.user_id` integer memerlukan mapping customer lama ke UUID; mengubah tipe kolom saja tidak merekonstruksi hubungan tersebut.
3. Jalankan `cmd/migrate` tanpa `-business`. Ini hanya menambahkan `point_go_tokens`, `point_go_revoked_tokens`, dan `point_go_locks` beserta lock singleton.
4. Pertahankan hostname/prefix `/api`, `APP_URL`, storage URL, dan CORS origin FE. Jalankan login, profile, list/pagination, form upload, orders, points, gift, dan address flows dari FE staging.
5. Secara default, gunakan JWT secret baru dan minta sesi login ulang. Token setup/verification/reset di cache Laravel tidak otomatis dipindahkan. `ACCEPT_LEGACY_TOKENS=true` hanya menerima HS256 Laravel dengan secret yang sama, expiry valid, dan provider hash `prv` yang sesuai. Blacklist Laravel tidak dibaca oleh Go. Jangan aktifkan penerimaan legacy tanpa mekanisme cutover/revocation yang menangani blacklist lama.
6. Jalankan server Go di belakang reverse proxy saat staging sudah cocok. Jika PHP dan Go menulis database secara bersamaan, pengaman concurrency pada Go tidak otomatis membuat writer PHP aman; cutover writer perlu dikoordinasikan.

Perbandingan staging dapat dijalankan tanpa mengirim mutation:

```bash
export LARAVEL_ADMIN_TOKEN='token-staging-laravel'
export GO_ADMIN_TOKEN='token-staging-go'
export LARAVEL_CUSTOMER_TOKEN='token-customer-staging-laravel'
export GO_CUSTOMER_TOKEN='token-customer-staging-go'
python3 scripts/compare_api.py --laravel https://api-laravel-staging.example --go https://api-go-staging.example
```

Script membandingkan status HTTP, keys, struktur JSON, dan nilai respons GET. URL base dan format timestamp dinormalisasi. Tanpa token terkait, endpoint protected dilewati. Kedua server harus membaca dataset yang sama atau snapshot yang ekuivalen. Perbandingan ini tidak menggantikan pengujian mutation dan alur FE. Hasil perbandingan live belum tersedia dalam paket ini.
