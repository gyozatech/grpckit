package grpckit

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfigFile(t *testing.T) {
	// Create temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
grpc:
  port: 9091
http:
  port: 8081
health:
  enabled: true
metrics:
  enabled: true
swagger:
  enabled: true
  path: /swagger/openapi.json
auth:
  protected_endpoints:
    - /api/v1/*
  public_endpoints:
    - /healthz
    - /readyz
log:
  level: debug
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadConfigFile(configPath)
	if err != nil {
		t.Fatalf("LoadConfigFile failed: %v", err)
	}

	if cfg.GRPC.Port != 9091 {
		t.Errorf("expected GRPC port 9091, got %d", cfg.GRPC.Port)
	}
	if cfg.HTTP.Port != 8081 {
		t.Errorf("expected HTTP port 8081, got %d", cfg.HTTP.Port)
	}
	if !cfg.Health.Enabled {
		t.Error("expected health enabled")
	}
	if !cfg.Metrics.Enabled {
		t.Error("expected metrics enabled")
	}
	if !cfg.Swagger.Enabled {
		t.Error("expected swagger enabled")
	}
	if cfg.Swagger.Path != "/swagger/openapi.json" {
		t.Errorf("expected swagger path /swagger/openapi.json, got %s", cfg.Swagger.Path)
	}
	if len(cfg.Auth.ProtectedEndpoints) != 1 || cfg.Auth.ProtectedEndpoints[0] != "/api/v1/*" {
		t.Errorf("unexpected protected endpoints: %v", cfg.Auth.ProtectedEndpoints)
	}
	if len(cfg.Auth.PublicEndpoints) != 2 {
		t.Errorf("expected 2 public endpoints, got %d", len(cfg.Auth.PublicEndpoints))
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("expected log level debug, got %s", cfg.Log.Level)
	}
}

func TestLoadConfigFile_NotFound(t *testing.T) {
	_, err := LoadConfigFile("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestLoadConfigFile_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	if err := os.WriteFile(configPath, []byte("invalid: yaml: content: ["), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := LoadConfigFile(configPath)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestApplyConfigFile(t *testing.T) {
	cfg := newServerConfig()
	fileCfg := &Config{
		GRPC:    GRPCConfig{Port: 9091},
		HTTP:    HTTPConfig{Port: 8081},
		Health:  FeatureConfig{Enabled: true},
		Metrics: FeatureConfig{Enabled: true},
		Swagger: SwaggerConfig{Enabled: true, Path: "/swagger.json"},
		Auth: AuthConfig{
			ProtectedEndpoints: []string{"/api/*"},
			PublicEndpoints:    []string{"/public/*"},
		},
		Log: LogConfig{Level: "debug"},
	}

	applyConfigFile(cfg, fileCfg)

	if cfg.grpcPort != 9091 {
		t.Errorf("expected gRPC port 9091, got %d", cfg.grpcPort)
	}
	if cfg.httpPort != 8081 {
		t.Errorf("expected HTTP port 8081, got %d", cfg.httpPort)
	}
	if !cfg.healthEnabled {
		t.Error("expected health enabled")
	}
	if !cfg.metricsEnabled {
		t.Error("expected metrics enabled")
	}
	if !cfg.swaggerEnabled {
		t.Error("expected swagger enabled")
	}
	if cfg.swaggerPath != "/swagger.json" {
		t.Errorf("expected swagger path /swagger.json, got %s", cfg.swaggerPath)
	}
	if len(cfg.protectedEndpoints) != 1 {
		t.Errorf("expected 1 protected endpoint, got %d", len(cfg.protectedEndpoints))
	}
	if len(cfg.publicEndpoints) != 1 {
		t.Errorf("expected 1 public endpoint, got %d", len(cfg.publicEndpoints))
	}
	if cfg.logLevel != "debug" {
		t.Errorf("expected log level debug, got %s", cfg.logLevel)
	}
}

func TestApplyConfigFile_ZeroValues(t *testing.T) {
	cfg := newServerConfig()
	originalGRPCPort := cfg.grpcPort
	originalHTTPPort := cfg.httpPort

	// Empty config should not override defaults
	fileCfg := &Config{}
	applyConfigFile(cfg, fileCfg)

	if cfg.grpcPort != originalGRPCPort {
		t.Errorf("gRPC port should not change with zero value, got %d", cfg.grpcPort)
	}
	if cfg.httpPort != originalHTTPPort {
		t.Errorf("HTTP port should not change with zero value, got %d", cfg.httpPort)
	}
}

func TestApplyEnvVars(t *testing.T) {
	// Save original env vars
	envVars := []string{
		"GRPCKIT_GRPC_PORT",
		"GRPCKIT_HTTP_PORT",
		"GRPCKIT_HEALTH_ENABLED",
		"GRPCKIT_METRICS_ENABLED",
		"GRPCKIT_SWAGGER_ENABLED",
		"GRPCKIT_SWAGGER_PATH",
		"GRPCKIT_LOG_LEVEL",
		"GRPCKIT_GRACEFUL_TIMEOUT",
		"GRPCKIT_PROTECTED_ENDPOINTS",
		"GRPCKIT_PUBLIC_ENDPOINTS",
	}
	originalValues := make(map[string]string)
	for _, key := range envVars {
		originalValues[key] = os.Getenv(key)
	}
	defer func() {
		for key, val := range originalValues {
			if val == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, val)
			}
		}
	}()

	// Set test env vars
	os.Setenv("GRPCKIT_GRPC_PORT", "9092")
	os.Setenv("GRPCKIT_HTTP_PORT", "8082")
	os.Setenv("GRPCKIT_HEALTH_ENABLED", "true")
	os.Setenv("GRPCKIT_METRICS_ENABLED", "1")
	os.Setenv("GRPCKIT_SWAGGER_ENABLED", "yes")
	os.Setenv("GRPCKIT_SWAGGER_PATH", "/api/swagger.json")
	os.Setenv("GRPCKIT_LOG_LEVEL", "warn")
	os.Setenv("GRPCKIT_GRACEFUL_TIMEOUT", "60s")
	os.Setenv("GRPCKIT_PROTECTED_ENDPOINTS", "/api/v1/*,/admin/*")
	os.Setenv("GRPCKIT_PUBLIC_ENDPOINTS", "/healthz,/readyz")

	cfg := newServerConfig()
	applyEnvVars(cfg)

	if cfg.grpcPort != 9092 {
		t.Errorf("expected gRPC port 9092, got %d", cfg.grpcPort)
	}
	if cfg.httpPort != 8082 {
		t.Errorf("expected HTTP port 8082, got %d", cfg.httpPort)
	}
	if !cfg.healthEnabled {
		t.Error("expected health enabled")
	}
	if !cfg.metricsEnabled {
		t.Error("expected metrics enabled")
	}
	if !cfg.swaggerEnabled {
		t.Error("expected swagger enabled")
	}
	if cfg.swaggerPath != "/api/swagger.json" {
		t.Errorf("expected swagger path /api/swagger.json, got %s", cfg.swaggerPath)
	}
	if cfg.logLevel != "warn" {
		t.Errorf("expected log level warn, got %s", cfg.logLevel)
	}
	if cfg.gracefulTimeout != 60*time.Second {
		t.Errorf("expected graceful timeout 60s, got %v", cfg.gracefulTimeout)
	}
	if len(cfg.protectedEndpoints) != 2 {
		t.Errorf("expected 2 protected endpoints, got %d", len(cfg.protectedEndpoints))
	}
	if len(cfg.publicEndpoints) != 2 {
		t.Errorf("expected 2 public endpoints, got %d", len(cfg.publicEndpoints))
	}
}

func TestApplyEnvVars_InvalidValues(t *testing.T) {
	os.Setenv("GRPCKIT_GRPC_PORT", "invalid")
	os.Setenv("GRPCKIT_GRACEFUL_TIMEOUT", "invalid")
	defer func() {
		os.Unsetenv("GRPCKIT_GRPC_PORT")
		os.Unsetenv("GRPCKIT_GRACEFUL_TIMEOUT")
	}()

	cfg := newServerConfig()
	originalGRPCPort := cfg.grpcPort
	originalTimeout := cfg.gracefulTimeout

	applyEnvVars(cfg)

	// Invalid values should be ignored
	if cfg.grpcPort != originalGRPCPort {
		t.Errorf("invalid gRPC port should be ignored, got %d", cfg.grpcPort)
	}
	if cfg.gracefulTimeout != originalTimeout {
		t.Errorf("invalid timeout should be ignored, got %v", cfg.gracefulTimeout)
	}
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"True", true},
		{"TRUE", true},
		{"1", true},
		{"yes", true},
		{"Yes", true},
		{"YES", true},
		{"on", true},
		{"On", true},
		{"ON", true},
		{"false", false},
		{"False", false},
		{"FALSE", false},
		{"0", false},
		{"no", false},
		{"off", false},
		{"", false},
		{"invalid", false},
		{"  true  ", true},
		{"  false  ", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseBool(tt.input)
			if result != tt.expected {
				t.Errorf("parseBool(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestWithConfigFile(t *testing.T) {
	// Create temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
grpc:
  port: 9093
http:
  port: 8083
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg := newServerConfig()
	opt := WithConfigFile(configPath)
	opt(cfg)

	if cfg.grpcPort != 9093 {
		t.Errorf("expected gRPC port 9093, got %d", cfg.grpcPort)
	}
	if cfg.httpPort != 8083 {
		t.Errorf("expected HTTP port 8083, got %d", cfg.httpPort)
	}
}

func TestWithConfigFile_NonExistent(t *testing.T) {
	cfg := newServerConfig()
	originalGRPCPort := cfg.grpcPort

	opt := WithConfigFile("/nonexistent/config.yaml")
	opt(cfg)

	// Should not fail, just ignore missing file
	if cfg.grpcPort != originalGRPCPort {
		t.Errorf("non-existent config should not modify config, got gRPC port %d", cfg.grpcPort)
	}
}
