package grpckit

import (
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the configuration file structure.
type Config struct {
	GRPC    GRPCConfig    `yaml:"grpc"`
	HTTP    HTTPConfig    `yaml:"http"`
	Health  FeatureConfig `yaml:"health"`
	Metrics FeatureConfig `yaml:"metrics"`
	Swagger SwaggerConfig `yaml:"swagger"`
	Auth    AuthConfig    `yaml:"auth"`
	Log     LogConfig     `yaml:"log"`
}

// GRPCConfig holds gRPC server configuration.
type GRPCConfig struct {
	Port int `yaml:"port"`
}

// HTTPConfig holds HTTP server configuration.
type HTTPConfig struct {
	Port int `yaml:"port"`
}

// FeatureConfig holds feature toggle configuration.
type FeatureConfig struct {
	Enabled bool `yaml:"enabled"`
}

// SwaggerConfig holds Swagger configuration.
type SwaggerConfig struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
}

// AuthConfig holds authentication configuration.
type AuthConfig struct {
	ProtectedEndpoints []string `yaml:"protected_endpoints"`
	PublicEndpoints    []string `yaml:"public_endpoints"`
}

// LogConfig holds logging configuration.
type LogConfig struct {
	Level string `yaml:"level"`
}

// LoadConfigFile loads configuration from a YAML file.
func LoadConfigFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// applyConfigFile applies configuration from a file to the server config.
func applyConfigFile(cfg *serverConfig, fileCfg *Config) {
	if fileCfg.GRPC.Port > 0 {
		cfg.grpcPort = fileCfg.GRPC.Port
	}
	if fileCfg.HTTP.Port > 0 {
		cfg.httpPort = fileCfg.HTTP.Port
	}
	if fileCfg.Health.Enabled {
		cfg.healthEnabled = true
	}
	if fileCfg.Metrics.Enabled {
		cfg.metricsEnabled = true
	}
	if fileCfg.Swagger.Enabled {
		cfg.swaggerEnabled = true
		cfg.swaggerPath = fileCfg.Swagger.Path
	}
	if len(fileCfg.Auth.ProtectedEndpoints) > 0 {
		cfg.protectedEndpoints = fileCfg.Auth.ProtectedEndpoints
	}
	if len(fileCfg.Auth.PublicEndpoints) > 0 {
		cfg.publicEndpoints = fileCfg.Auth.PublicEndpoints
	}
	if fileCfg.Log.Level != "" {
		cfg.logLevel = fileCfg.Log.Level
	}
}

// applyEnvVars applies configuration from environment variables.
func applyEnvVars(cfg *serverConfig) {
	if v := os.Getenv("GRPCKIT_GRPC_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.grpcPort = port
		}
	}

	if v := os.Getenv("GRPCKIT_HTTP_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.httpPort = port
		}
	}

	if v := os.Getenv("GRPCKIT_HEALTH_ENABLED"); v != "" {
		cfg.healthEnabled = parseBool(v)
	}

	if v := os.Getenv("GRPCKIT_METRICS_ENABLED"); v != "" {
		cfg.metricsEnabled = parseBool(v)
	}

	if v := os.Getenv("GRPCKIT_SWAGGER_ENABLED"); v != "" {
		cfg.swaggerEnabled = parseBool(v)
	}

	if v := os.Getenv("GRPCKIT_SWAGGER_PATH"); v != "" {
		cfg.swaggerPath = v
	}

	if v := os.Getenv("GRPCKIT_LOG_LEVEL"); v != "" {
		cfg.logLevel = v
	}

	if v := os.Getenv("GRPCKIT_GRACEFUL_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.gracefulTimeout = d
		}
	}

	if v := os.Getenv("GRPCKIT_PROTECTED_ENDPOINTS"); v != "" {
		cfg.protectedEndpoints = strings.Split(v, ",")
	}

	if v := os.Getenv("GRPCKIT_PUBLIC_ENDPOINTS"); v != "" {
		cfg.publicEndpoints = strings.Split(v, ",")
	}
}

// parseBool parses a boolean from common string representations.
func parseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "1" || s == "yes" || s == "on"
}

// WithConfigFile loads configuration from a YAML file.
// File configuration is applied first, then overridden by code options.
func WithConfigFile(path string) Option {
	return func(c *serverConfig) {
		fileCfg, err := LoadConfigFile(path)
		if err != nil {
			// Log warning but don't fail - file config is optional
			return
		}
		applyConfigFile(c, fileCfg)
	}
}
