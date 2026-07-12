// Package config provides centralized configuration management for GuestFlow.
// Configuration is loaded from environment variables using Viper with
// support for .env files. All config values have sensible defaults for development.
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config is the top-level configuration container for all application settings.
type Config struct {
	App       AppConfig       `mapstructure:"app"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Redis     RedisConfig     `mapstructure:"redis"`
	JWT       JWTConfig       `mapstructure:"jwt"`
	Server    ServerConfig    `mapstructure:"server"`
	WhatsApp  WhatsAppConfig  `mapstructure:"whatsapp"`
	Email     EmailConfig     `mapstructure:"email"`
	Storage   StorageConfig   `mapstructure:"storage"`
	Log       LogConfig       `mapstructure:"log"`
	CORS      CORSConfig      `mapstructure:"cors"`
	Tenant    TenantConfig    `mapstructure:"tenant"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
}

// AppConfig contains general application settings.
type AppConfig struct {
	Name    string `mapstructure:"name"`
	Env     string `mapstructure:"env"`
	Version string `mapstructure:"version"`
	Debug   bool   `mapstructure:"debug"`
}

// DatabaseConfig contains PostgreSQL connection settings.
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Name            string        `mapstructure:"name"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
}

// DSN builds a PostgreSQL connection string from the config values.
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}

// RedisConfig contains Redis connection settings.
type RedisConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Password     string `mapstructure:"password"`
	DB           int    `mapstructure:"db"`
	PoolSize     int    `mapstructure:"pool_size"`
	MinIdleConns int    `mapstructure:"min_idle_conns"`
	MaxRetries   int    `mapstructure:"max_retries"`
}

// Addr returns the Redis server address in host:port format.
func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// JWTConfig contains JWT token settings.
type JWTConfig struct {
	AccessSecret  string        `mapstructure:"access_secret"`
	RefreshSecret string        `mapstructure:"refresh_secret"`
	AccessTTL     time.Duration `mapstructure:"access_ttl"`
	RefreshTTL    time.Duration `mapstructure:"refresh_ttl"`
	Issuer        string        `mapstructure:"issuer"`
	Audience      string        `mapstructure:"audience"`

	// Secret is a legacy fallback for setups that still use JWT_SECRET.
	Secret string `mapstructure:"secret"`
}

// ServerConfig contains HTTP server settings.
type ServerConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

// ListenAddr returns the server listen address in host:port format.
func (s ServerConfig) ListenAddr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// WhatsAppConfig contains WhatsApp Business API settings.
type WhatsAppConfig struct {
	Enabled            bool   `mapstructure:"enabled"`
	APIURL             string `mapstructure:"api_url"`
	PhoneNumberID      string `mapstructure:"phone_number_id"`
	AccessToken        string `mapstructure:"access_token"`
	WebhookVerifyToken string `mapstructure:"webhook_verify_token"`
}

// EmailConfig contains SMTP email settings.
type EmailConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
	FromName string `mapstructure:"from_name"`
	UseTLS   bool   `mapstructure:"use_tls"`
}

// StorageConfig contains file storage settings.
type StorageConfig struct {
	Provider  string `mapstructure:"provider"`
	Endpoint  string `mapstructure:"endpoint"`
	Region    string `mapstructure:"region"`
	Bucket    string `mapstructure:"bucket"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	BaseURL   string `mapstructure:"base_url"`
	LocalPath string `mapstructure:"local_path"`
}

// LogConfig contains logging settings.
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// CORSConfig contains Cross-Origin Resource Sharing settings.
type CORSConfig struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
	AllowedMethods []string `mapstructure:"allowed_methods"`
	AllowedHeaders []string `mapstructure:"allowed_headers"`
}

// TenantConfig contains multi-tenancy settings.
type TenantConfig struct {
	Header           string `mapstructure:"header"`
	DefaultSubdomain string `mapstructure:"default_subdomain"`
}

// RateLimitConfig contains rate limiting settings.
type RateLimitConfig struct {
	RequestsPerSecond float64       `mapstructure:"requests_per_second"`
	Burst             int           `mapstructure:"burst"`
	TTL               time.Duration `mapstructure:"ttl"`
}

// ------------------------------------------------------------------------------
// Loading
// ------------------------------------------------------------------------------

// Load reads configuration from environment variables and optional .env file.
// It returns a fully populated Config struct with all values validated.
func Load() (*Config, error) {
	v := viper.New()

	// Enable environment variable binding
	v.AutomaticEnv()
	v.SetEnvPrefix("")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Try to load .env file if it exists (optional)
	v.SetConfigFile(".env")
	_ = v.ReadInConfig() // ignore error - .env is optional

	// Set defaults
	setDefaults(v)

	// Bind all environment variables
	bindEnvs(v)

	// Unmarshal into config struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate critical configuration
	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// setDefaults configures sensible development defaults for all settings.
func setDefaults(v *viper.Viper) {
	// App defaults
	v.SetDefault("app.name", "GuestFlow")
	v.SetDefault("app.env", "development")
	v.SetDefault("app.version", "1.0.0")
	v.SetDefault("app.debug", true)

	// Server defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", "30s")
	v.SetDefault("server.write_timeout", "30s")
	v.SetDefault("server.shutdown_timeout", "30s")

	// Database defaults
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "guestflow")
	v.SetDefault("database.password", "changeme")
	v.SetDefault("database.name", "guestflow")
	v.SetDefault("database.ssl_mode", "disable")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", "5m")
	v.SetDefault("database.conn_max_idle_time", "5m")

	// Redis defaults
	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.pool_size", 10)
	v.SetDefault("redis.min_idle_conns", 5)
	v.SetDefault("redis.max_retries", 3)

	// JWT defaults
	v.SetDefault("jwt.access_secret", "dev-access-secret-change-me")
	v.SetDefault("jwt.refresh_secret", "dev-refresh-secret-change-me")
	v.SetDefault("jwt.access_ttl", "15m")
	v.SetDefault("jwt.refresh_ttl", "168h")
	v.SetDefault("jwt.secret", "dev-secret-change-me")
	v.SetDefault("jwt.issuer", "guestflow")
	v.SetDefault("jwt.audience", "guestflow-api")

	// Rate limit defaults
	v.SetDefault("rate_limit.requests_per_second", 10.0)
	v.SetDefault("rate_limit.burst", 20)
	v.SetDefault("rate_limit.ttl", "60s")

	// WhatsApp defaults
	v.SetDefault("whatsapp.enabled", false)
	v.SetDefault("whatsapp.api_url", "https://graph.facebook.com/v18.0")

	// Email defaults
	v.SetDefault("email.enabled", false)
	v.SetDefault("email.port", 587)
	v.SetDefault("email.use_tls", true)

	// Storage defaults
	v.SetDefault("storage.provider", "local")
	v.SetDefault("storage.region", "us-east-1")
	v.SetDefault("storage.bucket", "guestflow-uploads")
	v.SetDefault("storage.local_path", "./uploads")

	// Log defaults
	v.SetDefault("log.level", "debug")
	v.SetDefault("log.format", "json")

	// CORS defaults
	v.SetDefault("cors.allowed_origins", []string{"*"})
	v.SetDefault("cors.allowed_methods", []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"})
	v.SetDefault("cors.allowed_headers", []string{"Content-Type", "Authorization", "X-Tenant-ID", "X-Request-ID"})

	// Tenant defaults
	v.SetDefault("tenant.header", "X-Tenant-ID")
	v.SetDefault("tenant.default_subdomain", "api")
}

// bindEnvs explicitly binds environment variables to config keys.
func bindEnvs(v *viper.Viper) {
	envBindings := []struct {
		key string
		env string
	}{
		{"app.name", "APP_NAME"},
		{"app.env", "APP_ENV"},
		{"app.version", "APP_VERSION"},
		{"app.debug", "APP_DEBUG"},
		{"server.host", "SERVER_HOST"},
		{"server.port", "SERVER_PORT"},
		{"server.read_timeout", "SERVER_READ_TIMEOUT"},
		{"server.write_timeout", "SERVER_WRITE_TIMEOUT"},
		{"server.shutdown_timeout", "SERVER_SHUTDOWN_TIMEOUT"},
		{"database.host", "DB_HOST"},
		{"database.port", "DB_PORT"},
		{"database.user", "DB_USER"},
		{"database.password", "DB_PASSWORD"},
		{"database.name", "DB_NAME"},
		{"database.ssl_mode", "DB_SSL_MODE"},
		{"database.max_open_conns", "DB_MAX_OPEN_CONNS"},
		{"database.max_idle_conns", "DB_MAX_IDLE_CONNS"},
		{"database.conn_max_lifetime", "DB_CONN_MAX_LIFETIME"},
		{"database.conn_max_idle_time", "DB_CONN_MAX_IDLE_TIME"},
		{"redis.host", "REDIS_HOST"},
		{"redis.port", "REDIS_PORT"},
		{"redis.password", "REDIS_PASSWORD"},
		{"redis.db", "REDIS_DB"},
		{"redis.pool_size", "REDIS_POOL_SIZE"},
		{"redis.min_idle_conns", "REDIS_MIN_IDLE_CONNS"},
		{"redis.max_retries", "REDIS_MAX_RETRIES"},
		{"jwt.access_secret", "JWT_ACCESS_SECRET"},
		{"jwt.refresh_secret", "JWT_REFRESH_SECRET"},
		{"jwt.access_ttl", "JWT_ACCESS_TTL"},
		{"jwt.refresh_ttl", "JWT_REFRESH_TTL"},
		{"jwt.secret", "JWT_SECRET"},
		{"jwt.issuer", "JWT_ISSUER"},
		{"jwt.audience", "JWT_AUDIENCE"},
		{"rate_limit.requests_per_second", "RATE_LIMIT_REQUESTS_PER_SECOND"},
		{"rate_limit.burst", "RATE_LIMIT_BURST"},
		{"rate_limit.ttl", "RATE_LIMIT_TTL"},
		{"whatsapp.enabled", "WHATSAPP_ENABLED"},
		{"whatsapp.api_url", "WHATSAPP_API_URL"},
		{"whatsapp.phone_number_id", "WHATSAPP_PHONE_NUMBER_ID"},
		{"whatsapp.access_token", "WHATSAPP_ACCESS_TOKEN"},
		{"whatsapp.webhook_verify_token", "WHATSAPP_WEBHOOK_VERIFY_TOKEN"},
		{"email.enabled", "EMAIL_ENABLED"},
		{"email.host", "EMAIL_HOST"},
		{"email.port", "EMAIL_PORT"},
		{"email.user", "EMAIL_USER"},
		{"email.password", "EMAIL_PASSWORD"},
		{"email.from", "EMAIL_FROM"},
		{"email.from_name", "EMAIL_FROM_NAME"},
		{"email.use_tls", "EMAIL_USE_TLS"},
		{"storage.provider", "STORAGE_PROVIDER"},
		{"storage.endpoint", "STORAGE_ENDPOINT"},
		{"storage.region", "STORAGE_REGION"},
		{"storage.bucket", "STORAGE_BUCKET"},
		{"storage.access_key", "STORAGE_ACCESS_KEY"},
		{"storage.secret_key", "STORAGE_SECRET_KEY"},
		{"storage.base_url", "STORAGE_BASE_URL"},
		{"storage.local_path", "STORAGE_LOCAL_PATH"},
		{"log.level", "LOG_LEVEL"},
		{"log.format", "LOG_FORMAT"},
		{"tenant.header", "TENANT_HEADER"},
		{"tenant.default_subdomain", "TENANT_DEFAULT_SUBDOMAIN"},
	}

	for _, binding := range envBindings {
		_ = v.BindEnv(binding.key, binding.env)
	}
}

// validate checks that critical configuration values are set correctly.
func validate(cfg *Config) error {
	if cfg.JWT.AccessSecret == "" {
		cfg.JWT.AccessSecret = cfg.JWT.Secret
	}
	if cfg.JWT.RefreshSecret == "" {
		cfg.JWT.RefreshSecret = cfg.JWT.Secret
	}

	if cfg.JWT.AccessSecret == "" {
		return fmt.Errorf("JWT_ACCESS_SECRET is required")
	}
	if cfg.JWT.RefreshSecret == "" {
		return fmt.Errorf("JWT_REFRESH_SECRET is required")
	}

	if cfg.App.Env == "production" {
		if cfg.JWT.AccessSecret == "dev-access-secret-change-me" || cfg.JWT.AccessSecret == "dev-secret-change-me" {
			return fmt.Errorf("JWT_ACCESS_SECRET must be set to a secure value in production")
		}
		if cfg.JWT.RefreshSecret == "dev-refresh-secret-change-me" || cfg.JWT.RefreshSecret == "dev-secret-change-me" {
			return fmt.Errorf("JWT_REFRESH_SECRET must be set to a secure value in production")
		}
	}

	if cfg.Database.Password == "" {
		return fmt.Errorf("DB_PASSWORD is required")
	}

	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		return fmt.Errorf("SERVER_PORT must be a valid port number (1-65535)")
	}

	return nil
}
