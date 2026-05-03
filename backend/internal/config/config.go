package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Auth     AuthConfig
	Email    EmailConfig
	Log      LogConfig
}

type ServerConfig struct {
	Port    string `mapstructure:"port"`
	Mode    string `mapstructure:"mode"` // debug | release
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
	SSLMode  string `mapstructure:"ssl_mode"`
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode,
	)
}

type AuthConfig struct {
	// For MVP: HS256 with a secret key (simpler, upgrade to RS256 before production)
	JWTSecret            string `mapstructure:"jwt_secret"`
	AccessTokenDuration  string `mapstructure:"access_token_duration"`  // e.g. "15m"
	RefreshTokenDuration string `mapstructure:"refresh_token_duration"` // e.g. "168h"
}

type EmailConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
	Enabled  bool   `mapstructure:"enabled"`
}

type LogConfig struct {
	Level string `mapstructure:"level"` // debug | info | warn | error
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./internal/config")
	viper.AddConfigPath(".")

	// Allow env vars to override config (e.g. DATABASE_HOST overrides database.host)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config: %w", err)
		}
		// Config file not found — rely on defaults + env vars
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	return &cfg, nil
}

func setDefaults() {
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.mode", "debug")

	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "sacramento")
	viper.SetDefault("database.password", "sacramento123")
	viper.SetDefault("database.name", "sacramento_finance")
	viper.SetDefault("database.ssl_mode", "disable")

	viper.SetDefault("auth.jwt_secret", "change-me-in-production-please")
	viper.SetDefault("auth.access_token_duration", "15m")
	viper.SetDefault("auth.refresh_token_duration", "168h")

	viper.SetDefault("email.host", "smtp.gmail.com")
	viper.SetDefault("email.port", 587)
	viper.SetDefault("email.from", "noreply@sacramento.finance")
	viper.SetDefault("email.enabled", false)

	viper.SetDefault("log.level", "info")
}
