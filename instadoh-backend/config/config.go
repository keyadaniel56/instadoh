package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	LND      LNDConfig
	Exchange ExchangeConfig
	JWT      JWTConfig
}

type ServerConfig struct {
	Host string
	Port int
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type LNDConfig struct {
	Host        string
	TLSCertPath string
	MacaroonPath string
}

type ExchangeConfig struct {
	APIKey     string
	BaseURL    string
	CacheTTL   int // seconds
}

type JWTConfig struct {
	Secret     string
	Expiration int // hours
}

func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.DBName,
		c.Database.SSLMode,
	)
}

func Load() *Config {
	// Load .env file if it exists (built-in loader, no external dependency)
	loadEnvFile()

	return &Config{
		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
			Port: getEnvInt("SERVER_PORT", 8080),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "instadoh"),
			Password: getEnv("DB_PASSWORD", "instadoh_secret"),
			DBName:   getEnv("DB_NAME", "instadoh"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		LND: LNDConfig{
			Host:         getEnv("LND_HOST", "localhost:10009"),
			TLSCertPath:  getEnv("LND_TLS_CERT", "/root/.lnd/tls.cert"),
			MacaroonPath: getEnv("LND_MACAROON", "/root/.lnd/data/chain/bitcoin/mainnet/admin.macaroon"),
		},
		Exchange: ExchangeConfig{
			APIKey:   getEnv("EXCHANGE_API_KEY", ""),
			BaseURL:  getEnv("EXCHANGE_BASE_URL", "https://api.exchangerate-api.com/v4/latest"),
			CacheTTL: getEnvInt("EXCHANGE_CACHE_TTL", 300),
		},
		JWT: JWTConfig{
			Secret:     getEnv("JWT_SECRET", "change-me-in-production"),
			Expiration: getEnvInt("JWT_EXPIRATION_HOURS", 24),
		},
	}
}

// loadEnvFile reads a .env file and sets environment variables.
// It looks for .env in the current working directory and parent directories.
func loadEnvFile() {
	dir, err := os.Getwd()
	if err != nil {
		return
	}

	// Search up the directory tree for .env
	for {
		envPath := filepath.Join(dir, ".env")
		if _, err := os.Stat(envPath); err == nil {
			readEnvFile(envPath)
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
}

func readEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split on first =
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove surrounding quotes if present
		value = strings.Trim(value, "\"'")

		// Only set if not already set in environment
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return fallback
}