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

type RabbitMQConfig struct {
	Host     string
	Port     string
	User     string
	Password string
}

type OTelConfig struct {
	Endpoint    string
	ServiceName string
}

type AppEnvConfig struct {
	serverCfg   *ServerConfig
	dbCfg       *DatabaseConfig
	rabbitmqCfg *RabbitMQConfig
	otelCfg     *OTelConfig
}

var _ config.ServerConfigProvider = (*AppEnvConfig)(nil)
var _ config.DatabaseConfigProvider = (*AppEnvConfig)(nil)
var _ config.RabbitMQConfigProvider = (*AppEnvConfig)(nil)
var _ config.OTelConfigProvider = (*AppEnvConfig)(nil)

func LoadConfig() (*AppEnvConfig, error) {
	serverCfg := &ServerConfig{
		Env:           getEnv("SERVER_ENV", "development"),
		Name:          getEnv("SERVER_NAME", "tianfu-account-service"),
		Host:          getEnv("SERVER_HOST", "0.0.0.0"),
		Port:          getEnv("SERVER_PORT", "8080"),
		BaseApiPrefix: getEnv("SERVER_BASEAPIPREFIX", "/api/v1"),
	}

	dbCfg := &DatabaseConfig{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", "password"),
		Name:     getEnv("DB_NAME", "account_db"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
		Timezone: getEnv("DB_TIMEZONE", "Asia/Bangkok"),
	}

	rabbitmqCfg := &RabbitMQConfig{
		Host:     getEnv("RABBITMQ_HOST", "localhost"),
		Port:     getEnv("RABBITMQ_PORT", "5672"),
		User:     getEnv("RABBITMQ_USER", "guest"),
		Password: getEnv("RABBITMQ_PASSWORD", "guest"),
	}

	otelCfg := &OTelConfig{
		Endpoint:    getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
		ServiceName: getEnv("OTEL_SERVICE_NAME", "tianfu-account-service"),
	}

	cfg := &AppEnvConfig{
		serverCfg:   serverCfg,
		dbCfg:       dbCfg,
		rabbitmqCfg: rabbitmqCfg,
		otelCfg:     otelCfg,
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
	if cfg.rabbitmqCfg.Host == "" || cfg.rabbitmqCfg.Port == "" {
		return fmt.Errorf("rabbitmq configuration is incomplete")
	}
	return nil
}

func (c *AppEnvConfig) GetServerEnv() string { return c.serverCfg.Env }
func (c *AppEnvConfig) GetServerName() string { return c.serverCfg.Name }
func (c *AppEnvConfig) GetServerHost() string { return c.serverCfg.Host }
func (c *AppEnvConfig) GetServerPort() string { return c.serverCfg.Port }
func (c *AppEnvConfig) GetServerBaseApiPrefix() string { return c.serverCfg.BaseApiPrefix }

func (c *AppEnvConfig) GetDBHost() string { return c.dbCfg.Host }
func (c *AppEnvConfig) GetDBPort() string { return c.dbCfg.Port }
func (c *AppEnvConfig) GetDBUser() string { return c.dbCfg.User }
func (c *AppEnvConfig) GetDBPassword() string { return c.dbCfg.Password }
func (c *AppEnvConfig) GetDBName() string { return c.dbCfg.Name }
func (c *AppEnvConfig) GetDBSSLMode() string { return c.dbCfg.SSLMode }
func (c *AppEnvConfig) GetDBTimezone() string { return c.dbCfg.Timezone }

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

func (c *AppEnvConfig) GetRabbitMQHost() string { return c.rabbitmqCfg.Host }
func (c *AppEnvConfig) GetRabbitMQPort() string { return c.rabbitmqCfg.Port }
func (c *AppEnvConfig) GetRabbitMQUser() string { return c.rabbitmqCfg.User }
func (c *AppEnvConfig) GetRabbitMQPassword() string { return c.rabbitmqCfg.Password }

func (c *AppEnvConfig) GetOTelEndpoint() string { return c.otelCfg.Endpoint }
func (c *AppEnvConfig) GetOTelServiceName() string { return c.otelCfg.ServiceName }
