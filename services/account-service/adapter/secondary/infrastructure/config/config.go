package config

import (
	"fmt"
	"os"

	"github.com/zhunismp/tianfu-bank/services/account-service/core/infrastructure/config"
)

type ServerConfig struct {
	Env           string
	Name          string
	Host          string
	Port          string
	BaseApiPrefix string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
	Timezone string
}

type PubsubConfig struct {
	Host string
	Port string
}

type AppEnvConfig struct {
	serverCfg *ServerConfig
	dbCfg     *DatabaseConfig
	pubsubCfg *PubsubConfig
}

var _ config.ServerConfigProvider = (*AppEnvConfig)(nil)
var _ config.DatabaseConfigProvider = (*AppEnvConfig)(nil)
var _ config.PubsubConfigProvider = (*AppEnvConfig)(nil)

func LoadConfig() (*AppEnvConfig, error) {
	serverCfg := &ServerConfig{
		Env:           getEnv("SERVER_ENV", "development"),
		Name:          getEnv("SERVER_NAME", "MyApp"),
		Host:          getEnv("SERVER_HOST", "0.0.0.0"),
		Port:          getEnv("SERVER_PORT", "8080"),
		BaseApiPrefix: getEnv("SERVER_BASEAPIPREFIX", "/api/v1"),
	}

	dbCfg := &DatabaseConfig{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", "1234"),
		Name:     getEnv("DB_NAME", "mydatabase"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
		Timezone: getEnv("DB_TIMEZONE", "Asia/Bangkok"),
	}

	pubsubCfg := &PubsubConfig{
		Host: getEnv("PUBSUB_HOST", "localhost"),
		Port: getEnv("PUBSUB_PORT", "6379"),
	}

	cfg := &AppEnvConfig{
		serverCfg: serverCfg,
		dbCfg:     dbCfg,
		pubsubCfg: pubsubCfg,
	}

	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("config validation error: %w", err)
	}

	return cfg, nil
}
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return defaultValue
}

func validateConfig(cfg *AppEnvConfig) error {
	if cfg.serverCfg.Port == "" {
		return fmt.Errorf("SERVER_PORT is required")
	}
	if cfg.dbCfg.Host == "" || cfg.dbCfg.Port == "" || cfg.dbCfg.User == "" || cfg.dbCfg.Password == "" || cfg.dbCfg.Name == "" {
		return fmt.Errorf("database configuration is incomplete")
	}
	if cfg.pubsubCfg.Host == "" || cfg.pubsubCfg.Port == "" {
		return fmt.Errorf("pubsub configuration is incomplete")
	}
	return nil
}

func (c *AppEnvConfig) GetServerEnv() string {
	return c.serverCfg.Env
}
func (c *AppEnvConfig) GetServerName() string {
	return c.serverCfg.Name
}
func (c *AppEnvConfig) GetServerHost() string {
	return c.serverCfg.Host
}
func (c *AppEnvConfig) GetServerPort() string {
	return c.serverCfg.Port
}
func (c *AppEnvConfig) GetServerBaseApiPrefix() string {
	return c.serverCfg.BaseApiPrefix
}

func (c *AppEnvConfig) GetDBHost() string {
	return c.dbCfg.Host
}
func (c *AppEnvConfig) GetDBPort() string {
	return c.dbCfg.Port
}
func (c *AppEnvConfig) GetDBUser() string {
	return c.dbCfg.User
}
func (c *AppEnvConfig) GetDBPassword() string {
	return c.dbCfg.Password
}
func (c *AppEnvConfig) GetDBName() string {
	return c.dbCfg.Name
}
func (c *AppEnvConfig) GetDBSSLMode() string {
	return c.dbCfg.SSLMode
}
func (c *AppEnvConfig) GetDBTimezone() string {
	return c.dbCfg.Timezone
}

func (c *AppEnvConfig) GetDBDSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		c.dbCfg.Host,
		c.dbCfg.Port,
		c.dbCfg.User,
		c.dbCfg.Password,
		c.dbCfg.Name,
		c.dbCfg.SSLMode,
		c.dbCfg.Timezone,
	)
}

func (c *AppEnvConfig) GetPubsubHost() string {
	return c.pubsubCfg.Host
}
func (c *AppEnvConfig) GetPubsubPort() string {
	return c.pubsubCfg.Port
}
