package database

import (
	"fmt"
	"log"
	"time"

	"instadoh-backend/config"
	"instadoh-backend/models"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Init establishes the database connection and runs auto-migrations
func Init(cfg *config.Config) *gorm.DB {
	var db *gorm.DB
	var err error

	if cfg.Database.Driver == "sqlite" {
		db, err = initSQLite(cfg)
	} else {
		db, err = initPostgres(cfg)
	}

	if err != nil {
		log.Fatalf("Could not connect to database: %v", err)
	}

	// Configure connection pool (only applies to postgres, sqlite ignores this)
	if cfg.Database.Driver != "sqlite" {
		sqlDB, err := db.DB()
		if err != nil {
			log.Fatalf("Failed to get underlying sql.DB: %v", err)
		}
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
		sqlDB.SetConnMaxLifetime(time.Hour)
	}

	// Run auto-migrations
	if err := runMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Seed countries
	if err := seedCountries(db); err != nil {
		log.Printf("Warning: Failed to seed countries: %v", err)
	}

	DB = db
	log.Printf("Database connected (%s) and migrations completed successfully", cfg.Database.Driver)
	return db
}

func initSQLite(cfg *config.Config) (*gorm.DB, error) {
	dsn := cfg.DSN() // returns file path like "instadoh.db"
	log.Printf("Connecting to SQLite database: %s", dsn)

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	// Enable WAL mode for better concurrent performance
	db.Exec("PRAGMA journal_mode=WAL")
	// Enable foreign keys
	db.Exec("PRAGMA foreign_keys=ON")

	return db, nil
}

func initPostgres(cfg *config.Config) (*gorm.DB, error) {
	dsn := cfg.DSN()
	log.Printf("Connecting to PostgreSQL database: %s:%d/%s", cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)

	var db *gorm.DB
	var err error

	// Retry connection with backoff
	for i := 0; i < 10; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})
		if err == nil {
			break
		}
		log.Printf("Failed to connect to database (attempt %d/10): %v", i+1, err)
		time.Sleep(time.Duration(i+1) * time.Second)
	}

	return db, err
}

func runMigrations(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.Country{},
		&models.Transaction{},
		&models.APIKey{},
		&models.MobileMoneyTransaction{},
		&models.CrossBorderTransaction{},
	)
}

func seedCountries(db *gorm.DB) error {
	countries := []models.Country{
		{Code: "US", Name: "United States", Currency: "USD", CurrencyName: "US Dollar", Flag: "🇺🇸"},
		{Code: "GB", Name: "United Kingdom", Currency: "GBP", CurrencyName: "British Pound", Flag: "🇬🇧"},
		{Code: "EU", Name: "European Union", Currency: "EUR", CurrencyName: "Euro", Flag: "🇪🇺"},
		{Code: "KE", Name: "Kenya", Currency: "KES", CurrencyName: "Kenyan Shilling", Flag: "🇰🇪"},
		{Code: "NG", Name: "Nigeria", Currency: "NGN", CurrencyName: "Nigerian Naira", Flag: "🇳🇬"},
		{Code: "ZA", Name: "South Africa", Currency: "ZAR", CurrencyName: "South African Rand", Flag: "🇿🇦"},
		{Code: "GH", Name: "Ghana", Currency: "GHS", CurrencyName: "Ghanaian Cedi", Flag: "🇬🇭"},
		{Code: "IN", Name: "India", Currency: "INR", CurrencyName: "Indian Rupee", Flag: "🇮🇳"},
		{Code: "BR", Name: "Brazil", Currency: "BRL", CurrencyName: "Brazilian Real", Flag: "🇧🇷"},
		{Code: "MX", Name: "Mexico", Currency: "MXN", CurrencyName: "Mexican Peso", Flag: "🇲🇽"},
		{Code: "JP", Name: "Japan", Currency: "JPY", CurrencyName: "Japanese Yen", Flag: "🇯🇵"},
		{Code: "CN", Name: "China", Currency: "CNY", CurrencyName: "Chinese Yuan", Flag: "🇨🇳"},
		{Code: "AU", Name: "Australia", Currency: "AUD", CurrencyName: "Australian Dollar", Flag: "🇦🇺"},
		{Code: "CA", Name: "Canada", Currency: "CAD", CurrencyName: "Canadian Dollar", Flag: "🇨🇦"},
		{Code: "CH", Name: "Switzerland", Currency: "CHF", CurrencyName: "Swiss Franc", Flag: "🇨🇭"},
		{Code: "SG", Name: "Singapore", Currency: "SGD", CurrencyName: "Singapore Dollar", Flag: "🇸🇬"},
		{Code: "PH", Name: "Philippines", Currency: "PHP", CurrencyName: "Philippine Peso", Flag: "🇵🇭"},
		{Code: "TZ", Name: "Tanzania", Currency: "TZS", CurrencyName: "Tanzanian Shilling", Flag: "🇹🇿"},
		{Code: "UG", Name: "Uganda", Currency: "UGX", CurrencyName: "Ugandan Shilling", Flag: "🇺🇬"},
		{Code: "ET", Name: "Ethiopia", Currency: "ETB", CurrencyName: "Ethiopian Birr", Flag: "🇪🇹"},
	}

	for _, c := range countries {
		var existing models.Country
		if err := db.Where("code = ?", c.Code).First(&existing).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := db.Create(&c).Error; err != nil {
					return fmt.Errorf("failed to seed country %s: %w", c.Code, err)
				}
			} else {
				return err
			}
		}
	}

	return nil
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	return DB
}