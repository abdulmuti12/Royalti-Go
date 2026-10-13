package platform

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	Address, Driver, DSN, AppURL, FrontendURL, Storage, Secret           string
	Origins                                                              []string
	TokenTTL                                                             time.Duration
	GoogleClientID, GoogleClientSecret                                   string
	MailHost, MailPort, MailUser, MailPassword, MailFrom, MailEncryption string
	AcceptLegacyTokens                                                   bool
}

func LoadEnv(path string) error {
	f, e := os.Open(path)
	if os.IsNotExist(e) {
		return nil
	}
	if e != nil {
		return e
	}
	defer f.Close()
	scan := bufio.NewScanner(f)
	for scan.Scan() {
		line := strings.TrimSpace(scan.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(strings.TrimPrefix(line, "export "), "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		if _, ok := os.LookupEnv(key); ok {
			continue
		}
		v := strings.TrimSpace(parts[1])
		if len(v) >= 2 && ((v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'')) {
			v = v[1 : len(v)-1]
		}
		if e = os.Setenv(key, v); e != nil {
			return e
		}
	}
	return scan.Err()
}
func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func LoadConfig() (Config, error) {
	if e := LoadEnv(".env"); e != nil {
		return Config{}, e
	}
	minutes, e := strconv.Atoi(env("JWT_TTL", "60"))
	if e != nil || minutes < 1 {
		return Config{}, fmt.Errorf("JWT_TTL must be positive minutes")
	}
	c := Config{Address: env("HTTP_ADDR", ":8000"), Driver: env("DB_CONNECTION", "mysql"), DSN: os.Getenv("DB_DSN"), AppURL: strings.TrimRight(env("APP_URL", "http://localhost:8000"), "/"), FrontendURL: strings.TrimRight(env("FRONTEND_URL", "http://localhost:3000"), "/"), Storage: env("STORAGE_PATH", "storage/app/public"), Secret: os.Getenv("JWT_SECRET"), TokenTTL: time.Duration(minutes) * time.Minute, GoogleClientID: os.Getenv("GOOGLE_CLIENT_ID"), GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"), MailHost: os.Getenv("MAIL_HOST"), MailEncryption: env("MAIL_ENCRYPTION", "tls"), MailPort: env("MAIL_PORT", "587"), MailUser: os.Getenv("MAIL_USERNAME"), MailPassword: os.Getenv("MAIL_PASSWORD"), MailFrom: env("MAIL_FROM_ADDRESS", "noreply@example.com"), AcceptLegacyTokens: env("ACCEPT_LEGACY_TOKENS", "false") == "true"}
	if len(c.Secret) < 32 {
		return c, fmt.Errorf("JWT_SECRET must contain at least 32 bytes")
	}
	for _, origin := range strings.Split(env("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:3001,http://localhost:3002,http://127.0.0.1:3000,http://127.0.0.1:3001,http://127.0.0.1:3002,http://localhost:8000,https://72.60.78.152,http://127.0.0.1,https://www.casaitalia-living.com,https://www.homelogystyle.com,https://homelogystyle.com,https://polflexoffice.com,https://api.polflexoffice.com,https://acces.casaitalia-living.com,https://acces.homelogystyle.com,https://acces.polflexoffice.com"), ",") {
		if v := strings.TrimSpace(origin); v != "" {
			c.Origins = append(c.Origins, v)
		}
	}
	if c.DSN == "" {
		if c.Driver == "sqlite" {
			c.DSN = env("DB_DATABASE", "point.db")
		} else {
			d := mysqlDriver.NewConfig()
			d.User = env("DB_USERNAME", "root")
			d.Passwd = os.Getenv("DB_PASSWORD")
			d.Net = "tcp"
			d.Addr = net.JoinHostPort(env("DB_HOST", "127.0.0.1"), env("DB_PORT", "3306"))
			d.DBName = env("DB_DATABASE", "api_point")
			d.ParseTime = true
			d.Loc = time.UTC
			d.Timeout = 10 * time.Second
			d.ReadTimeout = 15 * time.Second
			d.WriteTimeout = 15 * time.Second
			d.Params = map[string]string{"charset": "utf8mb4"}
			c.DSN = d.FormatDSN()
		}
	}
	return c, nil
}
func OpenDatabase(c Config) (*gorm.DB, error) {
	var dialector gorm.Dialector
	switch c.Driver {
	case "mysql":
		dialector = mysql.Open(c.DSN)
	case "sqlite":
		dialector = sqlite.Open(c.DSN)
	default:
		return nil, fmt.Errorf("unsupported DB_CONNECTION")
	}
	db, e := gorm.Open(dialector, &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if e != nil {
		return nil, e
	}
	sqlDB, e := db.DB()
	if e != nil {
		return nil, e
	}
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	if c.Driver == "sqlite" {
		sqlDB.SetMaxOpenConns(1)
		if e = db.Exec("PRAGMA foreign_keys = ON").Error; e != nil {
			return nil, e
		}
	}
	return db, nil
}
