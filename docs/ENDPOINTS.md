# Endpoint aktual

Acuan: repository API-Point, commit `0a6116c97a75da0e7cde7da83c9c75d81a30305b`. Prefix `/api` dipertahankan. PUT dan PATCH dicatat terpisah.

| Method | Path | Guard | Menu | Implementasi Laravel |
| --- | --- | --- | --- | --- |
| GET | `/api/check-route` | public | - | tersedia |
| POST | `/api/admins/login` | public | - | tersedia |
| GET | `/api/admins/check-route` | public | - | tersedia |
| POST | `/api/admins/forgot_password` | public | - | tersedia |
| POST | `/api/admins/reset_password` | public | - | tersedia |
| GET | `/api/admins/active-logo` | public | - | tersedia |
| GET | `/api/admins/me` | admin | - | tersedia |
| POST | `/api/admins/change-password` | admin | - | tersedia |
| GET | `/api/admins/admin` | admin | Admins | tersedia |
| POST | `/api/admins/admin` | admin | Admins | tersedia |
| GET | `/api/admins/admin/{id}` | admin | Admins | tersedia |
| PUT | `/api/admins/admin/{id}` | admin | Admins | tersedia |
| PATCH | `/api/admins/admin/{id}` | admin | Admins | tersedia |
| DELETE | `/api/admins/admin/{id}` | admin | Admins | tersedia |
| GET | `/api/admins/adminEdit/{id}` | admin | Admins | tersedia |
| GET | `/api/admins/role` | admin | Roles | tersedia |
| POST | `/api/admins/role` | admin | Roles | tersedia |
| GET | `/api/admins/role/{id}` | admin | Roles | tersedia |
| PUT | `/api/admins/role/{id}` | admin | Roles | tersedia |
| PATCH | `/api/admins/role/{id}` | admin | Roles | tersedia |
| DELETE | `/api/admins/role/{id}` | admin | Roles | tersedia |
| GET | `/api/admins/get-role` | admin | Roles | tersedia |
| GET | `/api/admins/get-menu` | admin | Roles | tersedia |
| GET | `/api/admins/get-edit-role/{id}` | admin | Roles | tersedia |
| POST | `/api/admins/logout` | admin | - | tersedia |
| GET | `/api/admins/promotion` | admin | Promotions | tersedia |
| POST | `/api/admins/promotion` | admin | Promotions | tersedia |
| GET | `/api/admins/promotion/{id}` | admin | Promotions | tersedia |
| PUT | `/api/admins/promotion/{id}` | admin | Promotions | tersedia |
| PATCH | `/api/admins/promotion/{id}` | admin | Promotions | tersedia |
| DELETE | `/api/admins/promotion/{id}` | admin | Promotions | tersedia |
| GET | `/api/admins/edit-promotion/{id}` | admin | Promotions | tersedia |
| GET | `/api/admins/banner` | admin | Banners | tersedia |
| POST | `/api/admins/banner` | admin | Banners | tersedia |
| GET | `/api/admins/banner/{id}` | admin | Banners | tersedia |
| PUT | `/api/admins/banner/{id}` | admin | Banners | tersedia |
| PATCH | `/api/admins/banner/{id}` | admin | Banners | tersedia |
| DELETE | `/api/admins/banner/{id}` | admin | Banners | tersedia |
| GET | `/api/admins/banner-edit/{id}` | admin | Banners | tersedia |
| GET | `/api/admins/contact-us` | admin | ContactUs | tersedia |
| POST | `/api/admins/contact-us` | admin | ContactUs | method belum tersedia, dilengkapi di Go |
| GET | `/api/admins/contact-us/{id}` | admin | ContactUs | tersedia |
| PUT | `/api/admins/contact-us/{id}` | admin | ContactUs | method belum tersedia, dilengkapi di Go |
| PATCH | `/api/admins/contact-us/{id}` | admin | ContactUs | method belum tersedia, dilengkapi di Go |
| DELETE | `/api/admins/contact-us/{id}` | admin | ContactUs | tersedia |
| GET | `/api/admins/list-contact` | admin | ContactUs | tersedia |
| POST | `/api/admins/list-contact` | admin | ContactUs | method belum tersedia, dilengkapi di Go |
| GET | `/api/admins/list-contact/{id}` | admin | ContactUs | tersedia |
| PUT | `/api/admins/list-contact/{id}` | admin | ContactUs | method belum tersedia, dilengkapi di Go |
| PATCH | `/api/admins/list-contact/{id}` | admin | ContactUs | method belum tersedia, dilengkapi di Go |
| DELETE | `/api/admins/list-contact/{id}` | admin | ContactUs | tersedia |
| GET | `/api/admins/customer` | admin | customer | tersedia |
| POST | `/api/admins/customer` | admin | customer | tersedia |
| GET | `/api/admins/customer/{id}` | admin | customer | tersedia |
| PUT | `/api/admins/customer/{id}` | admin | customer | method belum tersedia, dilengkapi di Go |
| PATCH | `/api/admins/customer/{id}` | admin | customer | method belum tersedia, dilengkapi di Go |
| DELETE | `/api/admins/customer/{id}` | admin | customer | method belum tersedia, dilengkapi di Go |
| GET | `/api/admins/get-orders` | admin | customer | tersedia |
| GET | `/api/admins/get-orders/{id}` | admin | customer | tersedia |
| GET | `/api/admins/get-customers` | admin | customer | tersedia |
| GET | `/api/admins/order` | admin | Orders | tersedia |
| POST | `/api/admins/order` | admin | Orders | tersedia |
| GET | `/api/admins/order/{id}` | admin | Orders | method belum tersedia, dilengkapi di Go |
| PUT | `/api/admins/order/{id}` | admin | Orders | method belum tersedia, dilengkapi di Go |
| PATCH | `/api/admins/order/{id}` | admin | Orders | method belum tersedia, dilengkapi di Go |
| DELETE | `/api/admins/order/{id}` | admin | Orders | method belum tersedia, dilengkapi di Go |
| GET | `/api/admins/get-categories` | admin | Orders | tersedia |
| GET | `/api/admins/get-brands` | admin | Orders | tersedia |
| GET | `/api/admins/transaction` | admin | Transactions | tersedia |
| POST | `/api/admins/transaction` | admin | Transactions | method belum tersedia, dilengkapi di Go |
| GET | `/api/admins/transaction/{id}` | admin | Transactions | method belum tersedia, dilengkapi di Go |
| PUT | `/api/admins/transaction/{id}` | admin | Transactions | method belum tersedia, dilengkapi di Go |
| PATCH | `/api/admins/transaction/{id}` | admin | Transactions | method belum tersedia, dilengkapi di Go |
| DELETE | `/api/admins/transaction/{id}` | admin | Transactions | method belum tersedia, dilengkapi di Go |
| GET | `/api/admins/brand` | admin | Brands | tersedia |
| POST | `/api/admins/brand` | admin | Brands | tersedia |
| GET | `/api/admins/brand/{id}` | admin | Brands | tersedia |
| PUT | `/api/admins/brand/{id}` | admin | Brands | tersedia |
| PATCH | `/api/admins/brand/{id}` | admin | Brands | tersedia |
| DELETE | `/api/admins/brand/{id}` | admin | Brands | tersedia |
| GET | `/api/admins/brand-edit/{id}` | admin | Brands | tersedia |
| GET | `/api/admins/dashboard` | admin | - | tersedia |
| GET | `/api/admins/category` | admin | Categories | tersedia |
| POST | `/api/admins/category` | admin | Categories | tersedia |
| GET | `/api/admins/category/{id}` | admin | Categories | tersedia |
| PUT | `/api/admins/category/{id}` | admin | Categories | tersedia |
| PATCH | `/api/admins/category/{id}` | admin | Categories | tersedia |
| DELETE | `/api/admins/category/{id}` | admin | Categories | tersedia |
| GET | `/api/admins/point` | admin | Points | tersedia |
| POST | `/api/admins/point` | admin | Points | tersedia |
| GET | `/api/admins/point/{id}` | admin | Points | tersedia |
| PUT | `/api/admins/point/{id}` | admin | Points | tersedia |
| PATCH | `/api/admins/point/{id}` | admin | Points | tersedia |
| DELETE | `/api/admins/point/{id}` | admin | Points | tersedia |
| GET | `/api/admins/point-edit/{id}` | admin | Points | tersedia |
| GET | `/api/admins/gift` | admin | Gifts | tersedia |
| POST | `/api/admins/gift` | admin | Gifts | tersedia |
| GET | `/api/admins/gift/{id}` | admin | Gifts | tersedia |
| PUT | `/api/admins/gift/{id}` | admin | Gifts | tersedia |
| PATCH | `/api/admins/gift/{id}` | admin | Gifts | tersedia |
| DELETE | `/api/admins/gift/{id}` | admin | Gifts | tersedia |
| GET | `/api/admins/gift-edit/{id}` | admin | Gifts | tersedia |
| GET | `/api/admins/gift-claims` | admin | - | tersedia |
| GET | `/api/admins/gift-claims/{id}` | admin | - | tersedia |
| POST | `/api/admins/gift-claims/{id}/approve` | admin | - | tersedia |
| POST | `/api/admins/gift-claims/{id}/reject` | admin | - | tersedia |
| POST | `/api/admins/gift-claims/{id}/complete` | admin | - | tersedia |
| GET | `/api/admins/member_level` | admin | - | tersedia |
| POST | `/api/admins/member_level` | admin | - | tersedia |
| GET | `/api/admins/member_level/{id}` | admin | - | tersedia |
| PUT | `/api/admins/member_level/{id}` | admin | - | tersedia |
| PATCH | `/api/admins/member_level/{id}` | admin | - | tersedia |
| DELETE | `/api/admins/member_level/{id}` | admin | - | tersedia |
| GET | `/api/admins/member_level-edit/{id}` | admin | - | tersedia |
| GET | `/api/admins/redeem-point` | admin | - | tersedia |
| POST | `/api/admins/redeem-point/calculate/{id}` | admin | - | tersedia |
| POST | `/api/admins/redeem-point/claim` | admin | - | tersedia |
| GET | `/api/admins/redeem-point/history/{id}` | admin | - | tersedia |
| GET | `/api/admins/redeem-point/{id}` | admin | - | tersedia |
| GET | `/api/admins/logo` | admin | - | tersedia |
| POST | `/api/admins/logo` | admin | - | tersedia |
| GET | `/api/admins/logo/{id}` | admin | - | tersedia |
| PUT | `/api/admins/logo/{id}` | admin | - | tersedia |
| PATCH | `/api/admins/logo/{id}` | admin | - | tersedia |
| DELETE | `/api/admins/logo/{id}` | admin | - | tersedia |
| GET | `/api/admins/logo-edit/{id}` | admin | - | tersedia |
| PATCH | `/api/admins/logo/{id}/set-active` | admin | - | tersedia |
| GET | `/api/admins/users` | admin | Users | tersedia |
| GET | `/api/admins/settings` | admin | Settings | tersedia |
| GET | `/api/part-color` | public | - | tersedia |
| GET | `/api/part-category` | public | - | tersedia |
| GET | `/api/part-type-product` | public | - | tersedia |
| GET | `/api/part-brand` | public | - | tersedia |
| GET | `/api/home-banner` | public | - | tersedia |
| GET | `/api/customer-promotion` | public | - | tersedia |
| POST | `/api/email-subscriber` | public | - | tersedia |
| POST | `/api/customers/register` | public | - | tersedia |
| POST | `/api/customers/social-register-auth` | public | - | tersedia |
| POST | `/api/customers/social-login` | public | - | tersedia |
| POST | `/api/customers/login-customer` | public | - | tersedia |
| POST | `/api/customers/set-password` | public | - | tersedia |
| POST | `/api/customers/resend-verification` | public | - | tersedia |
| POST | `/api/customers/verify-email` | public | - | tersedia |
| GET | `/api/customers/me` | customer | - | tersedia |
| POST | `/api/customers/logout` | customer | - | tersedia |
| POST | `/api/customers/change-password` | customer | - | tersedia |
| POST | `/api/customers/contact-us` | public | - | tersedia |
| POST | `/api/customers/email-subscriber` | public | - | tersedia |
| GET | `/api/customers/customer-data` | admin | customer | method belum tersedia, dilengkapi di Go |
| POST | `/api/customers/customer-data` | admin | customer | method belum tersedia, dilengkapi di Go |
| GET | `/api/customers/customer-data/{id}` | admin | customer | tersedia |
| PUT | `/api/customers/customer-data/{id}` | admin | customer | method belum tersedia, dilengkapi di Go |
| PATCH | `/api/customers/customer-data/{id}` | admin | customer | method belum tersedia, dilengkapi di Go |
| DELETE | `/api/customers/customer-data/{id}` | admin | customer | method belum tersedia, dilengkapi di Go |
| GET | `/api/customers/get-orders` | customer | - | tersedia |
| GET | `/api/customers/get-orders/{id}` | customer | - | tersedia |
| GET | `/api/customers/member-level` | customer | - | tersedia |
| GET | `/api/customers/gifts` | customer | - | tersedia |
| POST | `/api/customers/gifts/claim` | customer | - | tersedia |
| GET | `/api/customers/gifts/claims` | customer | - | tersedia |
| GET | `/api/customers/orders` | customer | - | method belum tersedia, dilengkapi di Go |
| GET | `/api/customers/orders/{id}` | customer | - | method belum tersedia, dilengkapi di Go |
| GET | `/api/customers/transactions` | public | - | tersedia |
| POST | `/api/customers/transactions` | public | - | tersedia |
| GET | `/api/customers/transactions/{id}` | public | - | tersedia |
| PUT | `/api/customers/transactions/{id}` | public | - | tersedia |
| PATCH | `/api/customers/transactions/{id}` | public | - | tersedia |
| DELETE | `/api/customers/transactions/{id}` | public | - | tersedia |
| GET | `/api/customers/get-province` | public | - | tersedia |
| POST | `/api/customers/get-city` | public | - | tersedia |
| POST | `/api/customers/get-districts` | public | - | tersedia |
| POST | `/api/customers/get-subdistrict` | public | - | tersedia |
| POST | `/api/customers/get-postal-code` | public | - | tersedia |
| GET | `/api/customers/get-addresses` | customer | - | tersedia |
| POST | `/api/customers/create-address` | customer | - | tersedia |
| POST | `/api/customers/delete-address` | customer | - | tersedia |
| GET | `/api/customers/get-address-texts` | customer | - | tersedia |
| POST | `/api/customers/create-address-text` | customer | - | tersedia |
| POST | `/api/customers/toggle-address-text-status` | customer | - | tersedia |
| POST | `/api/customers/update-address-text` | customer | - | tersedia |
| POST | `/api/customers/delete-address-text` | customer | - | tersedia |
