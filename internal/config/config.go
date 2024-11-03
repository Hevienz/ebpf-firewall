package config

import (
	"fmt"
	"log"
	"os"

	"github.com/danger-dream/ebpf-firewall/internal/utils"
	"github.com/spf13/viper"
)

type SecurityConfig struct {
	// Maximum number of errors allowed per IP before blocking
	IPErrorThreshold int `mapstructure:"ip-error-threshold"`
	// Time window in seconds for counting errors
	ErrorWindow int `mapstructure:"error-window"`
}

type RateLimitConfig struct {
	// Maximum number of requests allowed per interval
	RateLimitRequest int `mapstructure:"request"`
	// Rate limit time interval in seconds
	RateLimitInterval int `mapstructure:"interval"`
}

// Config holds all application configuration parameters
type Config struct {
	Version string `mapstructure:"version"`
	// API authentication token for securing the web interface
	Auth string `mapstructure:"auth"`
	// network interface name to monitor (e.g., eth0, ens33)
	Interface string `mapstructure:"interface"`
	// HTTP server listening address and port (e.g., :5678, 127.0.0.1:5678)
	Addr string `mapstructure:"addr"`

	// directory to store metrics data and blacklist
	DataDir string `mapstructure:"data-dir"`

	// file path to the MaxMind GeoLite2 City database
	GeoIPPath string `mapstructure:"geoip-path"`

	// interval to persist metrics data (in minutes), 0 means disable persistence
	MetricsPersistInterval int `mapstructure:"metrics-persist-interval"`

	// RetentionHours defines how long to keep packet data in storage (in hours)
	RetentionHours int `mapstructure:"retention-hours"`

	Security SecurityConfig `mapstructure:"security"`

	RateLimit RateLimitConfig `mapstructure:"rate-limit"`
}

var (
	appConfig Config
)

func Init() error {
	viper.SetDefault("version", "0.0.0")
	viper.SetDefault("auth", "")
	viper.SetDefault("interface", "")
	viper.SetDefault("addr", ":5678")
	viper.SetDefault("data-dir", "./data")
	viper.SetDefault("geoip-path", "GeoLite2-City.mmdb")
	viper.SetDefault("metrics-persist-interval", 10)
	viper.SetDefault("retention-hours", 720)
	viper.SetDefault("security.ip-error-threshold", 10)
	viper.SetDefault("security.error-window", 86400)
	viper.SetDefault("rate-limit.request", 120)
	viper.SetDefault("rate-limit.interval", 60)

	viper.SetConfigName("config")
	viper.AddConfigPath(".")

	viper.AutomaticEnv()
	viper.SetEnvPrefix("EBPF")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Printf("No config file found, using default values")
		} else {
			return fmt.Errorf("failed to read config: %w", err)
		}
	}

	if err := viper.Unmarshal(&appConfig); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := validateConfig(&appConfig); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	return nil
}

func validateConfig(config *Config) error {

	if config == nil {
		return fmt.Errorf("config is nil")
	}
	if config.Auth == "" {
		config.Auth = utils.GenerateRandomString(18)
		log.Printf("No auth token provided, generated random auth token: %s", config.Auth)
	}
	if config.Interface == "" {
		config.Interface = utils.GetDefaultInterface()
		log.Printf("No interface provided, using default interface: %s", config.Interface)
	} else if !utils.ValidateInterface(config.Interface) {
		return fmt.Errorf("invalid interface: %s", config.Interface)
	}

	if config.DataDir == "" {
		config.DataDir = "./data"
		log.Printf("No data directory provided, using default data directory: %s", config.DataDir)
	}

	if _, err := os.Stat(config.DataDir); os.IsNotExist(err) {
		if err := os.MkdirAll(config.DataDir, 0755); err != nil {
			return fmt.Errorf("failed to create data directory: %w", err)
		}
	}
	return nil
}

func GetConfig() *Config {
	return &appConfig
}
