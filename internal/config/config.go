package config

import (
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Redis    RedisConfig    `yaml:"redis"`
	SocketIO SocketIOConfig `yaml:"socketio"`
	Upload   UploadConfig   `yaml:"upload"`
	Logging  LoggingConfig  `yaml:"logging"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
	Env  string `yaml:"env"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// SocketIOConfig holds Socket.IO configuration
type SocketIOConfig struct {
	CORSOrigins  string        `yaml:"cors_origins"`
	PingTimeout  time.Duration `yaml:"ping_timeout"`
	PingInterval time.Duration `yaml:"ping_interval"`
}

// UploadConfig holds file upload configuration
type UploadConfig struct {
	MaxFileSize int64  `yaml:"max_file_size"`
	UploadDir   string `yaml:"upload_dir"`
	BaseURL     string `yaml:"base_url"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// Load loads configuration from config file and environment variables
func Load() (*Config, error) {
	cfg := &Config{}

	// Load from config file
	if err := loadFromFile(cfg); err != nil {
		logrus.WithError(err).Warn("Failed to load config file, using defaults")
	}

	// Override with environment variables
	loadFromEnv(cfg)

	// Validate configuration
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// loadFromFile loads configuration from YAML file
func loadFromFile(cfg *Config) error {
	file, err := os.Open("config.yaml")
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	return decoder.Decode(cfg)
}

// loadFromEnv loads configuration from environment variables
func loadFromEnv(cfg *Config) {
	if port := os.Getenv("PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Server.Port = p
		}
	}

	if host := os.Getenv("HOST"); host != "" {
		cfg.Server.Host = host
	}

	if env := os.Getenv("ENV"); env != "" {
		cfg.Server.Env = env
	}

	if redisAddr := os.Getenv("REDIS_ADDR"); redisAddr != "" {
		cfg.Redis.Addr = redisAddr
	}

	if redisPassword := os.Getenv("REDIS_PASSWORD"); redisPassword != "" {
		cfg.Redis.Password = redisPassword
	}

	if redisDB := os.Getenv("REDIS_DB"); redisDB != "" {
		if db, err := strconv.Atoi(redisDB); err == nil {
			cfg.Redis.DB = db
		}
	}

	if corsOrigins := os.Getenv("SOCKET_IO_CORS_ORIGINS"); corsOrigins != "" {
		cfg.SocketIO.CORSOrigins = corsOrigins
	}

	if maxFileSize := os.Getenv("MAX_FILE_SIZE"); maxFileSize != "" {
		if size, err := strconv.ParseInt(maxFileSize, 10, 64); err == nil {
			cfg.Upload.MaxFileSize = size
		}
	}

	if uploadDir := os.Getenv("UPLOAD_DIR"); uploadDir != "" {
		cfg.Upload.UploadDir = uploadDir
	}

	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		cfg.Logging.Level = logLevel
	}

	if logFormat := os.Getenv("LOG_FORMAT"); logFormat != "" {
		cfg.Logging.Format = logFormat
	}
}

// validate validates the configuration
func (c *Config) validate() error {
	if c.Server.Port <= 0 {
		c.Server.Port = 8080
	}

	if c.Server.Host == "" {
		c.Server.Host = "localhost"
	}

	if c.Server.Env == "" {
		c.Server.Env = "development"
	}

	if c.Redis.Addr == "" {
		c.Redis.Addr = "localhost:6379"
	}

	if c.SocketIO.CORSOrigins == "" {
		c.SocketIO.CORSOrigins = "*"
	}

	if c.SocketIO.PingTimeout == 0 {
		c.SocketIO.PingTimeout = 60 * time.Second
	}

	if c.SocketIO.PingInterval == 0 {
		c.SocketIO.PingInterval = 25 * time.Second
	}

	if c.Upload.MaxFileSize == 0 {
		c.Upload.MaxFileSize = 10485760 // 10MB
	}

	if c.Upload.UploadDir == "" {
		c.Upload.UploadDir = "uploads/"
	}

	if c.Upload.BaseURL == "" {
		c.Upload.BaseURL = "/uploads"
	}

	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}

	if c.Logging.Format == "" {
		c.Logging.Format = "json"
	}

	return nil
}

// GetServerAddress returns the server address
func (c *Config) GetServerAddress() string {
	return c.Server.Host + ":" + strconv.Itoa(c.Server.Port)
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Server.Env == "development"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.Server.Env == "production"
}
