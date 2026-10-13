package gormrepo

import "strings"
import "time"

// Persistence models use the existing Laravel table and column names.
// Migration is explicit; the HTTP server never calls AutoMigrate.

type Admin struct {
	ID            uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	Name          *string    `gorm:"column:name;size:255"`
	Email         *string    `gorm:"column:email;size:255;uniqueIndex"`
	Password      *string    `gorm:"column:password;type:text"`
	Status        *string    `gorm:"column:status;size:255"`
	RememberToken *string    `gorm:"column:remember_token;type:text"`
	LastLogin     *time.Time `gorm:"column:last_login"`
	CreatedAt     *time.Time `gorm:"column:created_at"`
	UpdatedAt     *time.Time `gorm:"column:updated_at"`
}

func (Admin) TableName() string { return "admins" }

type Role struct {
	ID        uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	Name      *string    `gorm:"column:name;size:255;uniqueIndex"`
	CreatedAt *time.Time `gorm:"column:created_at"`
	UpdatedAt *time.Time `gorm:"column:updated_at"`
}

func (Role) TableName() string { return "roles" }

type Menu struct {
	ID        uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	Name      *string    `gorm:"column:name;size:255"`
	Route     *string    `gorm:"column:route;size:255"`
	URL       *string    `gorm:"column:url;size:255"`
	CreatedAt *time.Time `gorm:"column:created_at"`
	UpdatedAt *time.Time `gorm:"column:updated_at"`
}

func (Menu) TableName() string { return "menus" }

type AdminRole struct {
	AdminID uint64 `gorm:"column:admin_id;primaryKey"`
	RoleID  uint64 `gorm:"column:role_id;primaryKey"`
}

func (AdminRole) TableName() string { return "admin_role" }

type RoleMenu struct {
	RoleID uint64 `gorm:"column:role_id;primaryKey"`
	MenuID uint64 `gorm:"column:menu_id;primaryKey"`
}

func (RoleMenu) TableName() string { return "role_menu" }

type Customer struct {
	ID              string     `gorm:"column:id;primaryKey;type:char(36)"`
	Name            *string    `gorm:"column:name;size:255"`
	FullName        *string    `gorm:"column:full_name;type:text"`
	Email           *string    `gorm:"column:email;size:255;uniqueIndex"`
	PhoneNumber     *string    `gorm:"column:phone_number;size:255;uniqueIndex"`
	Password        *string    `gorm:"column:password;type:text"`
	Status          *string    `gorm:"column:status;size:255"`
	Time            *time.Time `gorm:"column:time"`
	EmailVerifiedAt *time.Time `gorm:"column:email_verified_at"`
	Provider        *string    `gorm:"column:provider;size:255"`
	ProviderID      *string    `gorm:"column:provider_id;size:255"`
	Avatar          *string    `gorm:"column:avatar;type:text"`
	RememberToken   *string    `gorm:"column:remember_token;type:text"`
	Verify          *int64     `gorm:"column:verify"`
	CreatedAt       *time.Time `gorm:"column:created_at"`
	UpdatedAt       *time.Time `gorm:"column:updated_at"`
}

func (Customer) TableName() string { return "customers" }

type Category struct {
	ID          uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	Name        *string    `gorm:"column:name;size:255"`
	Description *string    `gorm:"column:description;type:text"`
	Image       *string    `gorm:"column:image;type:text"`
	Note        *string    `gorm:"column:note;type:text"`
	CreatedAt   *time.Time `gorm:"column:created_at"`
	UpdatedAt   *time.Time `gorm:"column:updated_at"`
}

func (Category) TableName() string { return "category" }

type Brand struct {
	ID              uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	Name            *string    `gorm:"column:name;size:255;uniqueIndex"`
	Description     *string    `gorm:"column:description;type:text"`
	Image           *string    `gorm:"column:image;type:text"`
	Note            *string    `gorm:"column:note;type:text"`
	CountryOfOrigin *string    `gorm:"column:country_of_origin;type:text"`
	Link            *string    `gorm:"column:link;type:text"`
	CreatedAt       *time.Time `gorm:"column:created_at"`
	UpdatedAt       *time.Time `gorm:"column:updated_at"`
}

func (Brand) TableName() string { return "brands" }

type Banner struct {
	ID          uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	Name        *string    `gorm:"column:name;size:255"`
	Description *string    `gorm:"column:description;type:text"`
	Note        *string    `gorm:"column:note;type:text"`
	Type        *string    `gorm:"column:type;type:text"`
	File        *string    `gorm:"column:file;type:text"`
	File2       *string    `gorm:"column:file2;type:text"`
	File3       *string    `gorm:"column:file3;type:text"`
	File4       *string    `gorm:"column:file4;type:text"`
	File5       *string    `gorm:"column:file5;type:text"`
	File6       *string    `gorm:"column:file6;type:text"`
	File7       *string    `gorm:"column:file7;type:text"`
	Note2       *string    `gorm:"column:note2;type:text"`
	Note3       *string    `gorm:"column:note3;type:text"`
	Note4       *string    `gorm:"column:note4;type:text"`
	Note5       *string    `gorm:"column:note5;type:text"`
	Title       *string    `gorm:"column:title;type:text"`
	Title2      *string    `gorm:"column:title2;type:text"`
	Title3      *string    `gorm:"column:title3;type:text"`
	Title4      *string    `gorm:"column:title4;type:text"`
	Title5      *string    `gorm:"column:title5;type:text"`
	Status      *string    `gorm:"column:status;size:255"`
	CreatedAt   *time.Time `gorm:"column:created_at"`
	UpdatedAt   *time.Time `gorm:"column:updated_at"`
}

func (Banner) TableName() string { return "banner" }

type Promotion struct {
	ID          uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	Name        *string    `gorm:"column:name;size:255"`
	Description *string    `gorm:"column:description;type:text"`
	Note        *string    `gorm:"column:note;type:text"`
	Type        *string    `gorm:"column:type;type:text"`
	File        *string    `gorm:"column:file;type:text"`
	File2       *string    `gorm:"column:file2;type:text"`
	File3       *string    `gorm:"column:file3;type:text"`
	File4       *string    `gorm:"column:file4;type:text"`
	File5       *string    `gorm:"column:file5;type:text"`
	Status      *string    `gorm:"column:status;size:255"`
	BrandID     *int64     `gorm:"column:brand_id;;index"`
	CreatedAt   *time.Time `gorm:"column:created_at"`
	UpdatedAt   *time.Time `gorm:"column:updated_at"`
}

func (Promotion) TableName() string { return "promotions" }

type ContactUs struct {
	ID          uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	Name        *string    `gorm:"column:name;size:255"`
	Email       *string    `gorm:"column:email;size:255"`
	PhoneNumber *string    `gorm:"column:phone_number;size:255"`
	Description *string    `gorm:"column:description;type:text"`
	CreatedAt   *time.Time `gorm:"column:created_at"`
	UpdatedAt   *time.Time `gorm:"column:updated_at"`
}

func (ContactUs) TableName() string { return "contact_us" }

type EmailSubscriber struct {
	ID        uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	Email     *string    `gorm:"column:email;size:255;uniqueIndex"`
	CreatedAt *time.Time `gorm:"column:created_at"`
	UpdatedAt *time.Time `gorm:"column:updated_at"`
}

func (EmailSubscriber) TableName() string { return "email_subscribers" }

type Logo struct {
	ID          uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	Description *string    `gorm:"column:description;type:text"`
	Image       *string    `gorm:"column:image;type:text"`
	IsActive    *int64     `gorm:"column:is_active"`
	CreatedAt   *time.Time `gorm:"column:created_at"`
	UpdatedAt   *time.Time `gorm:"column:updated_at"`
}

func (Logo) TableName() string { return "logos" }

type Point struct {
	ID          string     `gorm:"column:id;primaryKey;type:char(36)"`
	Name        *string    `gorm:"column:name;size:255"`
	Status      *string    `gorm:"column:status;size:255"`
	Description *string    `gorm:"column:description;type:text"`
	PricePoint  *int64     `gorm:"column:price_point"`
	RangePoint  *int64     `gorm:"column:range_point"`
	Point       *int64     `gorm:"column:point"`
	CreatedAt   *time.Time `gorm:"column:created_at"`
	UpdatedAt   *time.Time `gorm:"column:updated_at"`
}

func (Point) TableName() string { return "points" }

type Gift struct {
	ID          string     `gorm:"column:id;primaryKey;type:char(36)"`
	Name        *string    `gorm:"column:name;size:255"`
	Image       *string    `gorm:"column:image;type:text"`
	Description *string    `gorm:"column:description;type:text"`
	Status      *string    `gorm:"column:status;size:255"`
	TotalPoint  *int64     `gorm:"column:total_point"`
	CreatedAt   *time.Time `gorm:"column:created_at"`
	UpdatedAt   *time.Time `gorm:"column:updated_at"`
}

func (Gift) TableName() string { return "gifts" }

type MemberLevel struct {
	ID          string     `gorm:"column:id;primaryKey;type:char(36)"`
	Name        *string    `gorm:"column:name;size:255"`
	File        *string    `gorm:"column:file;type:text"`
	Description *string    `gorm:"column:description;type:text"`
	Status      *string    `gorm:"column:status;size:255"`
	MinPoint    *int64     `gorm:"column:min_point"`
	MaxPoint    *int64     `gorm:"column:max_point"`
	CreatedAt   *time.Time `gorm:"column:created_at"`
	UpdatedAt   *time.Time `gorm:"column:updated_at"`
}

func (MemberLevel) TableName() string { return "member_levels" }

type Order struct {
	ID              uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	OrderNumber     *string    `gorm:"column:order_number;size:255;uniqueIndex"`
	CustomerID      *string    `gorm:"column:customer_id;type:char(36);index"`
	RecipientName   *string    `gorm:"column:recipient_name;type:text"`
	Phone           *string    `gorm:"column:phone;type:text"`
	ShippingAddress *string    `gorm:"column:shipping_address;type:text"`
	ProvinceName    *string    `gorm:"column:province_name;type:text"`
	CityName        *string    `gorm:"column:city_name;type:text"`
	DistrictName    *string    `gorm:"column:district_name;type:text"`
	SubdistrictName *string    `gorm:"column:subdistrict_name;type:text"`
	PostalCode      *string    `gorm:"column:postal_code;type:text"`
	Status          *string    `gorm:"column:status;size:255"`
	Notes           *string    `gorm:"column:notes;type:text"`
	ProvinceID      *int64     `gorm:"column:province_id;;index"`
	CityID          *int64     `gorm:"column:city_id;;index"`
	DistrictID      *int64     `gorm:"column:district_id;;index"`
	SubdistrictID   *int64     `gorm:"column:subdistrict_id;;index"`
	Subtotal        *int64     `gorm:"column:subtotal"`
	ShippingCost    *int64     `gorm:"column:shipping_cost"`
	GrandTotal      *int64     `gorm:"column:grand_total"`
	CreatedAt       *time.Time `gorm:"column:created_at"`
	UpdatedAt       *time.Time `gorm:"column:updated_at"`
}

func (Order) TableName() string { return "orders" }

type OrderItem struct {
	ID          uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	ProductName *string    `gorm:"column:product_name;type:text"`
	Name        *string    `gorm:"column:name;size:255"`
	VariantName *string    `gorm:"column:variant_name;type:text"`
	Image       *string    `gorm:"column:image;type:text"`
	OrderID     *int64     `gorm:"column:order_id;;index"`
	ProductID   *int64     `gorm:"column:product_id;;index"`
	Quantity    *int64     `gorm:"column:quantity"`
	Price       *int64     `gorm:"column:price"`
	CategoryID  *int64     `gorm:"column:category_id;;index"`
	BrandID     *int64     `gorm:"column:brand_id;;index"`
	CreatedAt   *time.Time `gorm:"column:created_at"`
	UpdatedAt   *time.Time `gorm:"column:updated_at"`
}

func (OrderItem) TableName() string { return "order_items" }

type Transaction struct {
	ID              uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	TransactionCode *string    `gorm:"column:transaction_code;type:text"`
	Status          *string    `gorm:"column:status;size:255"`
	Note            *string    `gorm:"column:note;type:text"`
	OrderID         *int64     `gorm:"column:order_id;;index"`
	TotalPayment    *int64     `gorm:"column:total_payment"`
	AddressID       *int64     `gorm:"column:address_id;;index"`
	CreatedAt       *time.Time `gorm:"column:created_at"`
	UpdatedAt       *time.Time `gorm:"column:updated_at"`
}

func (Transaction) TableName() string { return "transaction" }

type Address struct {
	ID              uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	UserID          *string    `gorm:"column:user_id;type:text"`
	RecipientName   *string    `gorm:"column:recipient_name;type:text"`
	Phone           *string    `gorm:"column:phone;type:text"`
	ProvinceName    *string    `gorm:"column:province_name;type:text"`
	CityName        *string    `gorm:"column:city_name;type:text"`
	DistrictName    *string    `gorm:"column:district_name;type:text"`
	SubdistrictName *string    `gorm:"column:subdistrict_name;type:text"`
	PostalCode      *string    `gorm:"column:postal_code;type:text"`
	AddressLine     *string    `gorm:"column:address_line;type:text"`
	AddressDetail   *string    `gorm:"column:address_detail;type:text"`
	Province        *string    `gorm:"column:province;type:text"`
	City            *string    `gorm:"column:city;type:text"`
	District        *string    `gorm:"column:district;type:text"`
	ProvinceID      *int64     `gorm:"column:province_id;;index"`
	CityID          *int64     `gorm:"column:city_id;;index"`
	DistrictID      *int64     `gorm:"column:district_id;;index"`
	SubdistrictID   *int64     `gorm:"column:subdistrict_id;;index"`
	IsDefault       *int64     `gorm:"column:is_default"`
	CreatedAt       *time.Time `gorm:"column:created_at"`
	UpdatedAt       *time.Time `gorm:"column:updated_at"`
}

func (Address) TableName() string { return "addresses" }

type AddressText struct {
	ID         string     `gorm:"column:id;primaryKey;type:char(36)"`
	CustomerID *string    `gorm:"column:customer_id;type:char(36);index"`
	Address    *string    `gorm:"column:address;type:text"`
	Note       *string    `gorm:"column:note;type:text"`
	Status     *string    `gorm:"column:status;size:255"`
	CreatedAt  *time.Time `gorm:"column:created_at"`
	UpdatedAt  *time.Time `gorm:"column:updated_at"`
}

func (AddressText) TableName() string { return "address_text" }

type RedeemPoint struct {
	ID                string     `gorm:"column:id;primaryKey;type:char(36)"`
	CustomerID        *string    `gorm:"column:customer_id;type:char(36);index"`
	PointID           *string    `gorm:"column:point_id;type:text"`
	TotalTransaction  *int64     `gorm:"column:total_transaction"`
	TotalPointEarned  *int64     `gorm:"column:total_point_earned"`
	TotalPointClaimed *int64     `gorm:"column:total_point_claimed"`
	TotalPointActive  *int64     `gorm:"column:total_point_active"`
	TotalPointClosed  *int64     `gorm:"column:total_point_closed"`
	CreatedAt         *time.Time `gorm:"column:created_at"`
	UpdatedAt         *time.Time `gorm:"column:updated_at"`
}

func (RedeemPoint) TableName() string { return "redeem_points" }

type RedeemHistory struct {
	ID                string     `gorm:"column:id;primaryKey;type:char(36)"`
	RedeemPointID     *string    `gorm:"column:redeem_point_id;type:text"`
	CustomerID        *string    `gorm:"column:customer_id;type:char(36);index"`
	PointID           *string    `gorm:"column:point_id;type:text"`
	Type              *string    `gorm:"column:type;type:text"`
	Status            *string    `gorm:"column:status;size:255"`
	Description       *string    `gorm:"column:description;type:text"`
	PointAmount       *int64     `gorm:"column:point_amount"`
	PointBalance      *int64     `gorm:"column:point_balance"`
	TransactionAmount *int64     `gorm:"column:transaction_amount"`
	CreatedAt         *time.Time `gorm:"column:created_at"`
	UpdatedAt         *time.Time `gorm:"column:updated_at"`
}

func (RedeemHistory) TableName() string { return "redeem_histories" }

type GiftClaim struct {
	ID                     string     `gorm:"column:id;primaryKey;type:char(36)"`
	CustomerID             *string    `gorm:"column:customer_id;type:char(36);index"`
	GiftID                 *string    `gorm:"column:gift_id;type:text"`
	RedeemPointID          *string    `gorm:"column:redeem_point_id;type:text"`
	Status                 *string    `gorm:"column:status;size:255"`
	Notes                  *string    `gorm:"column:notes;type:text"`
	AdminNote              *string    `gorm:"column:admin_note;type:text"`
	ApprovedAt             *time.Time `gorm:"column:approved_at"`
	CompletedAt            *time.Time `gorm:"column:completed_at"`
	ApprovedBy             *int64     `gorm:"column:approved_by"`
	RequiredPoint          *int64     `gorm:"column:required_point"`
	CustomerPointAtRequest *int64     `gorm:"column:customer_point_at_request"`
	CreatedAt              *time.Time `gorm:"column:created_at"`
	UpdatedAt              *time.Time `gorm:"column:updated_at"`
}

func (GiftClaim) TableName() string { return "gift_claims" }

type AuditLog struct {
	ID          uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	UserID      *string    `gorm:"column:user_id;type:text"`
	UserType    *string    `gorm:"column:user_type;type:text"`
	UserName    *string    `gorm:"column:user_name;type:text"`
	Action      *string    `gorm:"column:action;type:text"`
	EntityType  *string    `gorm:"column:entity_type;type:text"`
	EntityID    *string    `gorm:"column:entity_id;type:text"`
	Description *string    `gorm:"column:description;type:text"`
	IpAddress   *string    `gorm:"column:ip_address;type:text"`
	UserAgent   *string    `gorm:"column:user_agent;type:text"`
	CreatedAt   *time.Time `gorm:"column:created_at"`
	UpdatedAt   *time.Time `gorm:"column:updated_at"`
}

func (AuditLog) TableName() string { return "audit_logs" }

type Province struct {
	ProvName *string `gorm:"column:prov_name;type:text"`
	ProvID   int64   `gorm:"column:prov_id;primaryKey;autoIncrement:false;index"`
}

func (Province) TableName() string { return "master_province" }

type City struct {
	CityName *string `gorm:"column:city_name;type:text"`
	CityID   int64   `gorm:"column:city_id;primaryKey;autoIncrement:false;index"`
	ProvID   *int64  `gorm:"column:prov_id;;index"`
}

func (City) TableName() string { return "master_city" }

type District struct {
	DisName *string `gorm:"column:dis_name;type:text"`
	DisID   int64   `gorm:"column:dis_id;primaryKey;autoIncrement:false;index"`
	CityID  *int64  `gorm:"column:city_id;;index"`
}

func (District) TableName() string { return "master_district" }

type Subdistrict struct {
	SubdisName *string `gorm:"column:subdis_name;type:text"`
	SubdisID   int64   `gorm:"column:subdis_id;primaryKey;autoIncrement:false;index"`
	DisID      *int64  `gorm:"column:dis_id;;index"`
}

func (Subdistrict) TableName() string { return "master_subdistrict" }

type PostalCode struct {
	PostalCode *string `gorm:"column:postal_code;type:text"`
	PostalID   int64   `gorm:"column:postal_id;primaryKey;autoIncrement:false;index"`
	SubdisID   *int64  `gorm:"column:subdis_id;;index"`
	DisID      *int64  `gorm:"column:dis_id;;index"`
	CityID     *int64  `gorm:"column:city_id;;index"`
	ProvID     *int64  `gorm:"column:prov_id;;index"`
}

func (PostalCode) TableName() string { return "master_postal_code" }

type TokenState struct {
	Key       string     `gorm:"column:key;primaryKey;size:255"`
	Hash      *string    `gorm:"column:hash;size:255"`
	Subject   *string    `gorm:"column:subject;size:255"`
	Kind      *string    `gorm:"column:kind;size:255"`
	ExpiresAt *time.Time `gorm:"column:expires_at"`
}

func (TokenState) TableName() string { return "point_go_tokens" }

type RevokedToken struct {
	Jti       string     `gorm:"column:jti;primaryKey;size:255"`
	ExpiresAt *time.Time `gorm:"column:expires_at"`
}

func (RevokedToken) TableName() string { return "point_go_revoked_tokens" }

type Lock struct {
	Key string `gorm:"column:key;primaryKey;size:255"`
}

func (Lock) TableName() string { return "point_go_locks" }

type Schema struct {
	Model            any
	UUID, Timestamps bool
	Columns          map[string]bool
}

func fields(s string) map[string]bool {
	m := map[string]bool{}
	for _, f := range strings.Fields(s) {
		m[f] = true
	}
	return m
}

var schemas = map[string]Schema{
	"admins":                  {Model: &Admin{}, UUID: false, Timestamps: true, Columns: fields("id name email password status remember_token last_login created_at updated_at")},
	"roles":                   {Model: &Role{}, UUID: false, Timestamps: true, Columns: fields("id name created_at updated_at")},
	"menus":                   {Model: &Menu{}, UUID: false, Timestamps: true, Columns: fields("id name route url created_at updated_at")},
	"admin_role":              {Model: &AdminRole{}, UUID: false, Timestamps: false, Columns: fields("admin_id role_id")},
	"role_menu":               {Model: &RoleMenu{}, UUID: false, Timestamps: false, Columns: fields("role_id menu_id")},
	"customers":               {Model: &Customer{}, UUID: true, Timestamps: true, Columns: fields("id name full_name email phone_number password status time email_verified_at provider provider_id avatar remember_token verify created_at updated_at")},
	"category":                {Model: &Category{}, UUID: false, Timestamps: true, Columns: fields("id name description image note created_at updated_at")},
	"brands":                  {Model: &Brand{}, UUID: false, Timestamps: true, Columns: fields("id name description image note country_of_origin link created_at updated_at")},
	"banner":                  {Model: &Banner{}, UUID: false, Timestamps: true, Columns: fields("id name description note type file file2 file3 file4 file5 file6 file7 note2 note3 note4 note5 title title2 title3 title4 title5 status created_at updated_at")},
	"promotions":              {Model: &Promotion{}, UUID: false, Timestamps: true, Columns: fields("id name description note type file file2 file3 file4 file5 status brand_id created_at updated_at")},
	"contact_us":              {Model: &ContactUs{}, UUID: false, Timestamps: true, Columns: fields("id name email phone_number description created_at updated_at")},
	"email_subscribers":       {Model: &EmailSubscriber{}, UUID: false, Timestamps: true, Columns: fields("id email created_at updated_at")},
	"logos":                   {Model: &Logo{}, UUID: false, Timestamps: true, Columns: fields("id description image is_active created_at updated_at")},
	"points":                  {Model: &Point{}, UUID: true, Timestamps: true, Columns: fields("id name status description price_point range_point point created_at updated_at")},
	"gifts":                   {Model: &Gift{}, UUID: true, Timestamps: true, Columns: fields("id name image description status total_point created_at updated_at")},
	"member_levels":           {Model: &MemberLevel{}, UUID: true, Timestamps: true, Columns: fields("id name file description status min_point max_point created_at updated_at")},
	"orders":                  {Model: &Order{}, UUID: false, Timestamps: true, Columns: fields("id order_number customer_id recipient_name phone shipping_address province_name city_name district_name subdistrict_name postal_code status notes province_id city_id district_id subdistrict_id subtotal shipping_cost grand_total created_at updated_at")},
	"order_items":             {Model: &OrderItem{}, UUID: false, Timestamps: true, Columns: fields("id product_name name variant_name image order_id product_id quantity price category_id brand_id created_at updated_at")},
	"transaction":             {Model: &Transaction{}, UUID: false, Timestamps: true, Columns: fields("id transaction_code status note order_id total_payment address_id created_at updated_at")},
	"addresses":               {Model: &Address{}, UUID: false, Timestamps: true, Columns: fields("id user_id recipient_name phone province_name city_name district_name subdistrict_name postal_code address_line address_detail province city district province_id city_id district_id subdistrict_id is_default created_at updated_at")},
	"address_text":            {Model: &AddressText{}, UUID: true, Timestamps: true, Columns: fields("id customer_id address note status created_at updated_at")},
	"redeem_points":           {Model: &RedeemPoint{}, UUID: true, Timestamps: true, Columns: fields("id customer_id point_id total_transaction total_point_earned total_point_claimed total_point_active total_point_closed created_at updated_at")},
	"redeem_histories":        {Model: &RedeemHistory{}, UUID: true, Timestamps: true, Columns: fields("id redeem_point_id customer_id point_id type status description point_amount point_balance transaction_amount created_at updated_at")},
	"gift_claims":             {Model: &GiftClaim{}, UUID: true, Timestamps: true, Columns: fields("id customer_id gift_id redeem_point_id status notes admin_note approved_at completed_at approved_by required_point customer_point_at_request created_at updated_at")},
	"audit_logs":              {Model: &AuditLog{}, UUID: false, Timestamps: true, Columns: fields("id user_id user_type user_name action entity_type entity_id description ip_address user_agent created_at updated_at")},
	"master_province":         {Model: &Province{}, UUID: false, Timestamps: false, Columns: fields("prov_name prov_id")},
	"master_city":             {Model: &City{}, UUID: false, Timestamps: false, Columns: fields("city_name city_id prov_id")},
	"master_district":         {Model: &District{}, UUID: false, Timestamps: false, Columns: fields("dis_name dis_id city_id")},
	"master_subdistrict":      {Model: &Subdistrict{}, UUID: false, Timestamps: false, Columns: fields("subdis_name subdis_id dis_id")},
	"master_postal_code":      {Model: &PostalCode{}, UUID: false, Timestamps: false, Columns: fields("postal_code postal_id subdis_id dis_id city_id prov_id")},
	"point_go_tokens":         {Model: &TokenState{}, UUID: false, Timestamps: false, Columns: fields("key hash subject kind expires_at")},
	"point_go_revoked_tokens": {Model: &RevokedToken{}, UUID: false, Timestamps: false, Columns: fields("jti expires_at")},
	"point_go_locks":          {Model: &Lock{}, UUID: false, Timestamps: false, Columns: fields("key")},
}
