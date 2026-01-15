#!/bin/bash
#
# create-service.sh - Generate a grpckit microservice boilerplate
#
# This script creates a complete project structure for a gRPC + REST
# microservice using grpckit library.
#
# Usage:
#   ./scripts/create-service.sh --name=<service-name> --module=<go-module-path> [--output=<dir>]
#   ./scripts/create-service.sh --name <service-name> --module <go-module-path> [--output <dir>]
#
# Example:
#   ./scripts/create-service.sh --name=user --module=github.com/myorg/user-service --output=./my-service
#   ./scripts/create-service.sh --proto=https://example.com/myservice.proto --module=github.com/myorg/my-service
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
OUTPUT_DIR=""
SERVICE_NAME=""
MODULE_PATH=""
PROTO_URL=""
GO_PACKAGE=""
GRPCKIT_VERSION=""
SWAGGER_URL=""
GRPC_PORT="9090"
HTTP_PORT="8080"

# Print usage
usage() {
    cat << EOF
Usage: $(basename "$0") [OPTIONS]

Generate a grpckit microservice boilerplate.

Required arguments:
  --name=<name>, -n <name>      Service name (e.g., "user", "order", "item")
                                Used for proto package, file names, and Go types
                                Not required if --proto is specified
  --module=<path>, -m <path>    Go module path (e.g., "github.com/myorg/user-service")

Optional arguments:
  --output=<dir>, -o <dir>      Output directory (default: ./<service-name>-service)
  --proto=<url>, -p <url>       URL or path to an existing proto file to use
                                If specified, reads the proto to extract service info
  --go-package=<path>           Go import path for the proto's generated code
                                Required if proto doesn't have go_package option
  --grpckit-version=<ver>       grpckit version (e.g., "v0.0.2")
                                If not specified, uses placeholder for go mod tidy
  --swagger=<url>               URL to swagger JSON file, fetched at build time
                                via 'make swagger' and embedded into the binary
  --grpc-port=<port>            gRPC port (default: 9090)
  --http-port=<port>            HTTP port (default: 8080)
  --help, -h                    Show this help message

Examples:
  $(basename "$0") --name=user --module=github.com/myorg/user-service
  $(basename "$0") -n order -m github.com/myorg/order-service -o ./services/order
  $(basename "$0") --name=item --module=github.com/myorg/item-service --output=./item-svc
  $(basename "$0") --proto=https://example.com/api.proto --module=github.com/myorg/api-service

EOF
    exit 1
}

# Print colored message
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Extract value from --arg=value or return empty
extract_value() {
    echo "$1" | sed 's/^[^=]*=//'
}

# Check if argument has = format
has_equals() {
    [[ "$1" == *"="* ]]
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --name=*)
                SERVICE_NAME=$(extract_value "$1")
                shift
                ;;
            --name|-n)
                SERVICE_NAME="$2"
                shift 2
                ;;
            --module=*)
                MODULE_PATH=$(extract_value "$1")
                shift
                ;;
            --module|-m)
                MODULE_PATH="$2"
                shift 2
                ;;
            --output=*)
                OUTPUT_DIR=$(extract_value "$1")
                shift
                ;;
            --output|-o)
                OUTPUT_DIR="$2"
                shift 2
                ;;
            --proto=*)
                PROTO_URL=$(extract_value "$1")
                shift
                ;;
            --proto|-p)
                PROTO_URL="$2"
                shift 2
                ;;
            --go-package=*)
                GO_PACKAGE=$(extract_value "$1")
                shift
                ;;
            --go-package)
                GO_PACKAGE="$2"
                shift 2
                ;;
            --grpckit-version=*)
                GRPCKIT_VERSION=$(extract_value "$1")
                shift
                ;;
            --grpckit-version)
                GRPCKIT_VERSION="$2"
                shift 2
                ;;
            --swagger=*)
                SWAGGER_URL=$(extract_value "$1")
                shift
                ;;
            --swagger)
                SWAGGER_URL="$2"
                shift 2
                ;;
            --grpc-port=*)
                GRPC_PORT=$(extract_value "$1")
                shift
                ;;
            --grpc-port)
                GRPC_PORT="$2"
                shift 2
                ;;
            --http-port=*)
                HTTP_PORT=$(extract_value "$1")
                shift
                ;;
            --http-port)
                HTTP_PORT="$2"
                shift 2
                ;;
            --help|-h)
                usage
                ;;
            *)
                print_error "Unknown argument: $1"
                usage
                ;;
        esac
    done

    # If proto URL is specified, extract service name from it if not provided
    if [[ -n "$PROTO_URL" && -z "$SERVICE_NAME" ]]; then
        # Extract filename from URL and remove .proto extension
        local proto_filename=$(basename "$PROTO_URL")
        SERVICE_NAME="${proto_filename%.proto}"
        print_info "Extracted service name from proto: $SERVICE_NAME"
    fi

    # Validate required arguments
    if [[ -z "$SERVICE_NAME" ]]; then
        print_error "Missing required argument: --name (or --proto to extract name automatically)"
        usage
    fi

    if [[ -z "$MODULE_PATH" ]]; then
        print_error "Missing required argument: --module"
        usage
    fi

    # Validate service name (alphanumeric and underscores only)
    if [[ ! "$SERVICE_NAME" =~ ^[a-z][a-z0-9_]*$ ]]; then
        print_error "Service name must start with a lowercase letter and contain only lowercase letters, numbers, and underscores"
        exit 1
    fi

    # Set default output directory if not specified
    if [[ -z "$OUTPUT_DIR" ]]; then
        OUTPUT_DIR="./${SERVICE_NAME}-service"
    fi
}

# Convert service name to different cases
# Works on both GNU (Linux) and BSD (macOS) systems
to_pascal_case() {
    local input="$1"
    local result=""
    local capitalize_next=true

    for (( i=0; i<${#input}; i++ )); do
        char="${input:$i:1}"
        if [[ "$char" == "_" ]]; then
            capitalize_next=true
        elif $capitalize_next; then
            # Convert to uppercase
            result+=$(echo "$char" | tr '[:lower:]' '[:upper:]')
            capitalize_next=false
        else
            result+="$char"
        fi
    done

    echo "$result"
}

to_camel_case() {
    local pascal=$(to_pascal_case "$1")
    local first_char=$(echo "${pascal:0:1}" | tr '[:upper:]' '[:lower:]')
    echo "${first_char}${pascal:1}"
}

# Create directory structure
create_directories() {
    print_info "Creating directory structure in ${OUTPUT_DIR}..."

    mkdir -p "${OUTPUT_DIR}"
    mkdir -p "${OUTPUT_DIR}/internal/service"

    # Only create proto directory if we're generating a proto file
    if [[ -z "$PROTO_URL" ]]; then
        mkdir -p "${OUTPUT_DIR}/proto/gen"
    fi

    # Create api directory if swagger is enabled
    if [[ -n "$SWAGGER_URL" ]]; then
        mkdir -p "${OUTPUT_DIR}/api"
    fi

    print_success "Directory structure created"
}

# Fetch proto content (to temp file for parsing, not copying)
fetch_proto_content() {
    print_info "Reading proto file from ${PROTO_URL}..."

    TEMP_PROTO_FILE=$(mktemp)

    if [[ "$PROTO_URL" == http* ]]; then
        # Download from URL
        if command -v curl &> /dev/null; then
            curl -fsSL "$PROTO_URL" -o "$TEMP_PROTO_FILE"
        elif command -v wget &> /dev/null; then
            wget -q "$PROTO_URL" -O "$TEMP_PROTO_FILE"
        else
            print_error "Neither curl nor wget found. Please install one of them."
            exit 1
        fi
    else
        # Read from local path
        if [[ -f "$PROTO_URL" ]]; then
            cat "$PROTO_URL" > "$TEMP_PROTO_FILE"
        else
            print_error "Proto file not found: $PROTO_URL"
            exit 1
        fi
    fi

    print_success "Proto file loaded"
}

# Extract service info from proto file
extract_proto_info() {
    # Extract service name from proto file
    PROTO_SERVICE_NAME=$(grep -E "^service\s+\w+" "$TEMP_PROTO_FILE" | head -1 | awk '{print $2}' || echo "")

    # Extract package name (trim whitespace)
    PROTO_PACKAGE=$(grep -E "^package\s+" "$TEMP_PROTO_FILE" | head -1 | sed 's/package[[:space:]]*//; s/;.*//' | tr -d '[:space:]' || echo "${SERVICE_NAME}.v1")

    # Use provided --go-package or extract from proto
    if [[ -n "$GO_PACKAGE" ]]; then
        PROTO_GO_PACKAGE="$GO_PACKAGE"
        print_info "Using provided go_package: ${PROTO_GO_PACKAGE}"
    else
        # Extract go_package if present (get the value between quotes)
        PROTO_GO_PACKAGE=$(grep -E "option\s+go_package\s*=" "$TEMP_PROTO_FILE" | head -1 | grep -oE '"[^"]+"' | tr -d '"' || echo "")

        if [[ -n "$PROTO_GO_PACKAGE" ]]; then
            print_info "Detected go_package: ${PROTO_GO_PACKAGE}"
        fi
    fi

    # Validate we have a go_package for external proto
    if [[ -z "$PROTO_GO_PACKAGE" ]]; then
        print_error "Could not find go_package in proto file and --go-package not provided"
        print_error "Please specify --go-package=<import-path> for the proto's generated Go code"
        rm -f "$TEMP_PROTO_FILE"
        exit 1
    fi

    # Extract import path (before semicolon if format is "path;alias")
    PROTO_GO_PACKAGE=$(echo "$PROTO_GO_PACKAGE" | sed 's/;.*//')

    if [[ -z "$PROTO_SERVICE_NAME" ]]; then
        print_warning "Could not extract service name from proto file. Using: $(to_pascal_case "$SERVICE_NAME")Service"
        PROTO_SERVICE_NAME="$(to_pascal_case "$SERVICE_NAME")Service"
    fi

    print_info "Detected proto service: ${PROTO_SERVICE_NAME}"
    print_info "Detected proto package: ${PROTO_PACKAGE}"

    # Clean up temp file
    rm -f "$TEMP_PROTO_FILE"
}

# Generate go.mod
generate_go_mod() {
    print_info "Generating go.mod..."

    local grpckit_version="${GRPCKIT_VERSION:-v0.0.0}"
    local version_comment=""

    if [[ -z "$GRPCKIT_VERSION" ]]; then
        version_comment=$'\n// TODO: Run \'go mod tidy\' to fetch the latest versions'
    fi

    cat > "${OUTPUT_DIR}/go.mod" << EOF
module ${MODULE_PATH}

go 1.22

require (
	github.com/gyozatech/grpckit ${grpckit_version}
	google.golang.org/grpc v1.60.0
	google.golang.org/protobuf v1.32.0
)
${version_comment}
EOF

    print_success "go.mod generated"
}

# Generate proto file
generate_proto() {
    local service_pascal=$(to_pascal_case "$SERVICE_NAME")

    print_info "Generating proto/${SERVICE_NAME}.proto..."

    cat > "${OUTPUT_DIR}/proto/${SERVICE_NAME}.proto" << EOF
syntax = "proto3";

package ${SERVICE_NAME}.v1;

option go_package = "${MODULE_PATH}/proto/gen;${SERVICE_NAME}pb";

import "google/api/annotations.proto";
import "protoc-gen-openapiv2/options/annotations.proto";

// OpenAPI documentation
option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_swagger) = {
  info: {
    title: "${service_pascal} Service API"
    version: "1.0"
    description: "API for ${service_pascal} service"
  }
  schemes: HTTP
  schemes: HTTPS
  consumes: "application/json"
  produces: "application/json"
};

// ${service_pascal}Service provides CRUD operations for ${SERVICE_NAME}s.
service ${service_pascal}Service {
  // Create${service_pascal} creates a new ${SERVICE_NAME}.
  rpc Create${service_pascal}(Create${service_pascal}Request) returns (Create${service_pascal}Response) {
    option (google.api.http) = {
      post: "/api/v1/${SERVICE_NAME}s"
      body: "*"
    };
  }

  // Get${service_pascal} retrieves a ${SERVICE_NAME} by ID.
  rpc Get${service_pascal}(Get${service_pascal}Request) returns (Get${service_pascal}Response) {
    option (google.api.http) = {
      get: "/api/v1/${SERVICE_NAME}s/{id}"
    };
  }

  // List${service_pascal}s retrieves all ${SERVICE_NAME}s.
  rpc List${service_pascal}s(List${service_pascal}sRequest) returns (List${service_pascal}sResponse) {
    option (google.api.http) = {
      get: "/api/v1/${SERVICE_NAME}s"
    };
  }

  // Update${service_pascal} updates an existing ${SERVICE_NAME}.
  rpc Update${service_pascal}(Update${service_pascal}Request) returns (Update${service_pascal}Response) {
    option (google.api.http) = {
      put: "/api/v1/${SERVICE_NAME}s/{id}"
      body: "*"
    };
  }

  // Delete${service_pascal} deletes a ${SERVICE_NAME} by ID.
  rpc Delete${service_pascal}(Delete${service_pascal}Request) returns (Delete${service_pascal}Response) {
    option (google.api.http) = {
      delete: "/api/v1/${SERVICE_NAME}s/{id}"
    };
  }
}

// ${service_pascal} represents a ${SERVICE_NAME} entity.
message ${service_pascal} {
  string id = 1;
  string name = 2;
  string description = 3;
  int64 created_at = 4;
  int64 updated_at = 5;
}

// Create${service_pascal}Request is the request for Create${service_pascal}.
message Create${service_pascal}Request {
  string name = 1;
  string description = 2;
}

// Create${service_pascal}Response is the response for Create${service_pascal}.
message Create${service_pascal}Response {
  ${service_pascal} ${SERVICE_NAME} = 1;
}

// Get${service_pascal}Request is the request for Get${service_pascal}.
message Get${service_pascal}Request {
  string id = 1;
}

// Get${service_pascal}Response is the response for Get${service_pascal}.
message Get${service_pascal}Response {
  ${service_pascal} ${SERVICE_NAME} = 1;
}

// List${service_pascal}sRequest is the request for List${service_pascal}s.
message List${service_pascal}sRequest {
  int32 page_size = 1;
  string page_token = 2;
}

// List${service_pascal}sResponse is the response for List${service_pascal}s.
message List${service_pascal}sResponse {
  repeated ${service_pascal} ${SERVICE_NAME}s = 1;
  string next_page_token = 2;
}

// Update${service_pascal}Request is the request for Update${service_pascal}.
message Update${service_pascal}Request {
  string id = 1;
  string name = 2;
  string description = 3;
}

// Update${service_pascal}Response is the response for Update${service_pascal}.
message Update${service_pascal}Response {
  ${service_pascal} ${SERVICE_NAME} = 1;
}

// Delete${service_pascal}Request is the request for Delete${service_pascal}.
message Delete${service_pascal}Request {
  string id = 1;
}

// Delete${service_pascal}Response is the response for Delete${service_pascal}.
message Delete${service_pascal}Response {
  bool success = 1;
}
EOF

    print_success "Proto file generated"
}

# Generate buf.yaml
generate_buf_yaml() {
    print_info "Generating proto/buf.yaml..."

    cat > "${OUTPUT_DIR}/proto/buf.yaml" << 'EOF'
version: v1
deps:
  - buf.build/googleapis/googleapis
  - buf.build/grpc-ecosystem/grpc-gateway
breaking:
  use:
    - FILE
lint:
  use:
    - DEFAULT
EOF

    print_success "buf.yaml generated"
}

# Generate buf.yaml for external proto (with instructions)
generate_buf_yaml_for_external_proto() {
    print_info "Generating proto/buf.yaml..."

    cat > "${OUTPUT_DIR}/proto/buf.yaml" << EOF
version: v1
deps:
  - buf.build/googleapis/googleapis
  - buf.build/grpc-ecosystem/grpc-gateway
  # TODO: Add your proto dependency here. Examples:
  #
  # From Buf Schema Registry:
  #   - buf.build/your-org/your-protos
  #
  # From Git repository:
  #   - buf.build/your-org/your-repo
  #
  # Proto source: ${PROTO_URL}
breaking:
  use:
    - FILE
lint:
  use:
    - DEFAULT
EOF

    print_success "buf.yaml generated (configure your proto dependency)"
}

# Generate buf.gen.yaml
generate_buf_gen_yaml() {
    print_info "Generating proto/buf.gen.yaml..."

    cat > "${OUTPUT_DIR}/proto/buf.gen.yaml" << 'EOF'
version: v1
managed:
  enabled: true
plugins:
  - plugin: buf.build/protocolbuffers/go
    out: gen
    opt: paths=source_relative
  - plugin: buf.build/grpc/go
    out: gen
    opt: paths=source_relative
  - plugin: buf.build/grpc-ecosystem/gateway
    out: gen
    opt:
      - paths=source_relative
      - generate_unbound_methods=true
  - plugin: buf.build/grpc-ecosystem/openapiv2
    out: gen
    opt:
      - allow_merge=true
      - merge_file_name=swagger
EOF

    print_success "buf.gen.yaml generated"
}

# Generate service implementation stub (for external proto)
generate_service_stub() {
    local service_pascal=$(to_pascal_case "$SERVICE_NAME")
    local actual_service_name="${PROTO_SERVICE_NAME:-${service_pascal}Service}"

    print_info "Generating service stub for ${actual_service_name}..."

    cat > "${OUTPUT_DIR}/internal/service/${SERVICE_NAME}_service.go" << EOF
package service

import (
	// "context"

	pb "${PROTO_GO_PACKAGE}"
)

// ${actual_service_name}Impl implements the ${actual_service_name} gRPC service.
// TODO: Implement all the RPC methods defined in your proto file.
type ${actual_service_name}Impl struct {
	pb.Unimplemented${actual_service_name}Server
}

// New${actual_service_name} creates a new ${actual_service_name} instance.
func New${actual_service_name}() *${actual_service_name}Impl {
	return &${actual_service_name}Impl{}
}

// TODO: Implement the RPC methods from your proto file.
// NOTE: Uncomment "context" import above when implementing methods.
// Example:
//
// func (s *${actual_service_name}Impl) YourMethod(ctx context.Context, req *pb.YourRequest) (*pb.YourResponse, error) {
//     // Your implementation here
//     return &pb.YourResponse{}, nil
// }
EOF

    print_success "Service stub generated"
}

# Generate service implementation
generate_service() {
    local service_pascal=$(to_pascal_case "$SERVICE_NAME")

    print_info "Generating internal/service/${SERVICE_NAME}_service.go..."

    cat > "${OUTPUT_DIR}/internal/service/${SERVICE_NAME}_service.go" << EOF
package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gyozatech/grpckit"
	pb "${MODULE_PATH}/proto/gen"
)

// ${service_pascal}Service implements the ${service_pascal}Service gRPC service.
type ${service_pascal}Service struct {
	pb.Unimplemented${service_pascal}ServiceServer
	mu    sync.RWMutex
	store map[string]*pb.${service_pascal}
}

// New${service_pascal}Service creates a new ${service_pascal}Service instance.
func New${service_pascal}Service() *${service_pascal}Service {
	return &${service_pascal}Service{
		store: make(map[string]*pb.${service_pascal}),
	}
}

// Create${service_pascal} creates a new ${SERVICE_NAME}.
func (s *${service_pascal}Service) Create${service_pascal}(ctx context.Context, req *pb.Create${service_pascal}Request) (*pb.Create${service_pascal}Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	${SERVICE_NAME} := &pb.${service_pascal}{
		Id:          fmt.Sprintf("%d.%06d", now.Unix(), now.Nanosecond()/1000),
		Name:        req.Name,
		Description: req.Description,
		CreatedAt:   now.Unix(),
		UpdatedAt:   now.Unix(),
	}

	s.store[${SERVICE_NAME}.Id] = ${SERVICE_NAME}

	return &pb.Create${service_pascal}Response{
		${service_pascal}: ${SERVICE_NAME},
	}, nil
}

// Get${service_pascal} retrieves a ${SERVICE_NAME} by ID.
func (s *${service_pascal}Service) Get${service_pascal}(ctx context.Context, req *pb.Get${service_pascal}Request) (*pb.Get${service_pascal}Response, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	${SERVICE_NAME}, ok := s.store[req.Id]
	if !ok {
		return nil, grpckit.ErrNotFound
	}

	return &pb.Get${service_pascal}Response{
		${service_pascal}: ${SERVICE_NAME},
	}, nil
}

// List${service_pascal}s retrieves all ${SERVICE_NAME}s.
func (s *${service_pascal}Service) List${service_pascal}s(ctx context.Context, req *pb.List${service_pascal}sRequest) (*pb.List${service_pascal}sResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	${SERVICE_NAME}s := make([]*pb.${service_pascal}, 0, len(s.store))
	for _, ${SERVICE_NAME} := range s.store {
		${SERVICE_NAME}s = append(${SERVICE_NAME}s, ${SERVICE_NAME})
	}

	return &pb.List${service_pascal}sResponse{
		${service_pascal}s: ${SERVICE_NAME}s,
	}, nil
}

// Update${service_pascal} updates an existing ${SERVICE_NAME}.
func (s *${service_pascal}Service) Update${service_pascal}(ctx context.Context, req *pb.Update${service_pascal}Request) (*pb.Update${service_pascal}Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	${SERVICE_NAME}, ok := s.store[req.Id]
	if !ok {
		return nil, grpckit.ErrNotFound
	}

	${SERVICE_NAME}.Name = req.Name
	${SERVICE_NAME}.Description = req.Description
	${SERVICE_NAME}.UpdatedAt = time.Now().Unix()

	return &pb.Update${service_pascal}Response{
		${service_pascal}: ${SERVICE_NAME},
	}, nil
}

// Delete${service_pascal} deletes a ${SERVICE_NAME} by ID.
func (s *${service_pascal}Service) Delete${service_pascal}(ctx context.Context, req *pb.Delete${service_pascal}Request) (*pb.Delete${service_pascal}Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.store[req.Id]; !ok {
		return nil, grpckit.ErrNotFound
	}

	delete(s.store, req.Id)

	return &pb.Delete${service_pascal}Response{
		Success: true,
	}, nil
}
EOF

    print_success "Service implementation generated"
}

# Generate main.go for external proto
generate_main_for_external_proto() {
    local service_pascal=$(to_pascal_case "$SERVICE_NAME")
    local actual_service_name="${PROTO_SERVICE_NAME:-${service_pascal}Service}"

    # Build swagger section based on whether SWAGGER_URL is provided
    local swagger_line

    if [[ -n "$SWAGGER_URL" ]]; then
        swagger_line="		// --- Swagger UI ---
		// Swagger spec fetched and embedded at build time via 'make swagger'
		// Source: ${SWAGGER_URL}
		// NOTE: Ensure the swagger version matches your imported proto version
		grpckit.WithSwagger(\"${SWAGGER_URL}\"),"
    else
        swagger_line="		// --- Swagger UI ---
		// Serves Swagger UI at /swagger/ with your OpenAPI spec
		// grpckit.WithSwagger(\"https://example.com/swagger.json\"),"
    fi

    print_info "Generating main.go for ${actual_service_name}..."

    cat > "${OUTPUT_DIR}/main.go" << EOF
package main

import (
	// "context"
	"log"
	"time"

	"github.com/gyozatech/grpckit"
	"${MODULE_PATH}/internal/service"
	pb "${PROTO_GO_PACKAGE}"
	"google.golang.org/grpc"
)


func main() {
	log.Println("Starting ${SERVICE_NAME} service...")

	// =========================================================================
	// Authentication Function (Optional)
	// =========================================================================
	// Uncomment and customize this function to enable authentication.
	// The function receives the Bearer token and should return an enriched
	// context with user information, or an error if authentication fails.
	// NOTE: Uncomment "context" import above when enabling authentication.
	//
	// authFunc := func(ctx context.Context, token string) (context.Context, error) {
	// 	if token == "" {
	// 		return nil, grpckit.ErrUnauthorized
	// 	}
	// 	// TODO: Validate token (JWT, database lookup, etc.)
	// 	// Example: userID, err := validateToken(token)
	// 	// if err != nil {
	// 	// 	return nil, grpckit.ErrUnauthorized
	// 	// }
	// 	return context.WithValue(ctx, "user_id", "user-from-token"), nil
	// }

	if err := grpckit.Run(
		// =====================================================================
		// Required: Service Registration
		// =====================================================================
		// Register your gRPC service implementation
		grpckit.WithGRPCService(func(s grpc.ServiceRegistrar) {
			pb.Register${actual_service_name}Server(s, service.New${actual_service_name}())
		}),

		// Register the REST gateway (grpc-gateway)
		grpckit.WithRESTService(pb.Register${actual_service_name}HandlerFromEndpoint),

		// =====================================================================
		// Server Ports
		// =====================================================================
		grpckit.WithGRPCPort(${GRPC_PORT}),
		grpckit.WithHTTPPort(${HTTP_PORT}),

		// Uncomment to run gRPC and HTTP on the same port (h2c multiplexing):
		// grpckit.WithGRPCPort(8080),
		// grpckit.WithHTTPPort(8080),

		// =====================================================================
		// Basic Features (Enabled)
		// =====================================================================
		grpckit.WithHealthCheck(),              // /healthz and /readyz endpoints
		grpckit.WithGracefulShutdown(30*time.Second),

		// =====================================================================
		// Optional Features (Uncomment to enable)
		// =====================================================================

		// --- Metrics ---
		// Prometheus metrics endpoint at /metrics
		// grpckit.WithMetrics(),

${swagger_line}

		// --- CORS ---
		// Enable Cross-Origin Resource Sharing for browser requests
		// grpckit.WithCORS(),
		//
		// Or with custom configuration:
		// grpckit.WithCORSConfig(grpckit.CORSConfig{
		// 	AllowedOrigins:   []string{"https://example.com"},
		// 	AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		// 	AllowedHeaders:   []string{"Authorization", "Content-Type"},
		// 	AllowCredentials: true,
		// 	MaxAge:           3600,
		// }),

		// --- Authentication ---
		// Uncomment WithAuth and configure public/protected endpoints
		// grpckit.WithAuth(authFunc),
		//
		// Option 1: Define public endpoints (everything else requires auth)
		// grpckit.WithPublicEndpoints(
		// 	"/healthz",
		// 	"/readyz",
		// 	"/metrics",
		// 	"/swagger/*",
		// ),
		//
		// Option 2: Define protected endpoints (everything else is public)
		// grpckit.WithProtectedEndpoints(
		// 	"/api/v1/${SERVICE_NAME}s/*",
		// ),

		// --- Custom Content Types ---
		// Enable form-urlencoded input (HTML forms)
		// grpckit.WithFormURLEncodedSupport(),
		//
		// Enable XML input/output
		// grpckit.WithXMLSupport(),
		//
		// Enable binary data (file downloads)
		// grpckit.WithBinarySupport(),
		//
		// Enable multipart form data (file uploads)
		// grpckit.WithMultipartSupport(),
		//
		// Enable plain text
		// grpckit.WithTextSupport(),

		// --- JSON Options ---
		// Customize JSON serialization
		// grpckit.WithJSONOptions(grpckit.JSONOptions{
		// 	UseProtoNames:   true,  // Use snake_case instead of camelCase
		// 	EmitUnpopulated: true,  // Include fields with zero values
		// 	Indent:          "  ",  // Pretty print
		// }),

		// --- gRPC Interceptors ---
		// Add custom unary interceptor
		// grpckit.WithUnaryInterceptor(func(
		// 	ctx context.Context,
		// 	req interface{},
		// 	info *grpc.UnaryServerInfo,
		// 	handler grpc.UnaryHandler,
		// ) (interface{}, error) {
		// 	log.Printf("[gRPC] %s", info.FullMethod)
		// 	return handler(ctx, req)
		// }),
		//
		// Add interceptor with endpoint exclusions
		// grpckit.WithUnaryInterceptor(myInterceptor,
		// 	grpckit.ExceptEndpoints("/package.Service/Method"),
		// ),
		//
		// Add stream interceptor
		// grpckit.WithStreamInterceptor(func(
		// 	srv interface{},
		// 	ss grpc.ServerStream,
		// 	info *grpc.StreamServerInfo,
		// 	handler grpc.StreamHandler,
		// ) error {
		// 	log.Printf("[gRPC Stream] %s", info.FullMethod)
		// 	return handler(srv, ss)
		// }),

		// --- HTTP Middleware ---
		// Add custom HTTP middleware (applies to all HTTP requests)
		// grpckit.WithHTTPMiddleware(func(next http.Handler) http.Handler {
		// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 		log.Printf("[HTTP] %s %s", r.Method, r.URL.Path)
		// 		next.ServeHTTP(w, r)
		// 	})
		// }),

		// --- Custom HTTP Handlers ---
		// Add custom HTTP endpoint (not exposed via gRPC)
		// grpckit.WithHTTPHandlerFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		// 	w.WriteHeader(http.StatusOK)
		// 	w.Write([]byte(\`{"status": "received"}\`))
		// }),

	); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
EOF

    print_success "main.go generated"
}

# Generate main.go
generate_main() {
    local service_pascal=$(to_pascal_case "$SERVICE_NAME")

    # Build swagger section based on whether SWAGGER_URL is provided
    local swagger_line

    if [[ -n "$SWAGGER_URL" ]]; then
        swagger_line="		// --- Swagger UI ---
		// Swagger spec fetched and embedded at build time via 'make swagger'
		// Source: ${SWAGGER_URL}
		// NOTE: Ensure the swagger version matches your imported proto version
		grpckit.WithSwagger(\"${SWAGGER_URL}\"),"
    else
        swagger_line="		// --- Swagger UI ---
		// Serves Swagger UI at /swagger/ with your OpenAPI spec
		// grpckit.WithSwagger(\"https://example.com/swagger.json\"),"
    fi

    print_info "Generating main.go..."

    cat > "${OUTPUT_DIR}/main.go" << EOF
package main

import (
	// "context"
	"log"
	"time"

	"github.com/gyozatech/grpckit"
	"${MODULE_PATH}/internal/service"
	pb "${MODULE_PATH}/proto/gen"
	"google.golang.org/grpc"
)

func main() {
	log.Println("Starting ${SERVICE_NAME} service...")

	// =========================================================================
	// Authentication Function (Optional)
	// =========================================================================
	// Uncomment and customize this function to enable authentication.
	// The function receives the Bearer token and should return an enriched
	// context with user information, or an error if authentication fails.
	// NOTE: Uncomment "context" import above when enabling authentication.
	//
	// authFunc := func(ctx context.Context, token string) (context.Context, error) {
	// 	if token == "" {
	// 		return nil, grpckit.ErrUnauthorized
	// 	}
	// 	// TODO: Validate token (JWT, database lookup, etc.)
	// 	// Example: userID, err := validateToken(token)
	// 	// if err != nil {
	// 	// 	return nil, grpckit.ErrUnauthorized
	// 	// }
	// 	return context.WithValue(ctx, "user_id", "user-from-token"), nil
	// }

	if err := grpckit.Run(
		// =====================================================================
		// Required: Service Registration
		// =====================================================================
		// Register your gRPC service implementation
		grpckit.WithGRPCService(func(s grpc.ServiceRegistrar) {
			pb.Register${service_pascal}ServiceServer(s, service.New${service_pascal}Service())
		}),

		// Register the REST gateway (grpc-gateway)
		grpckit.WithRESTService(pb.Register${service_pascal}ServiceHandlerFromEndpoint),

		// =====================================================================
		// Server Ports
		// =====================================================================
		grpckit.WithGRPCPort(${GRPC_PORT}),
		grpckit.WithHTTPPort(${HTTP_PORT}),

		// Uncomment to run gRPC and HTTP on the same port (h2c multiplexing):
		// grpckit.WithGRPCPort(8080),
		// grpckit.WithHTTPPort(8080),

		// =====================================================================
		// Basic Features (Enabled)
		// =====================================================================
		grpckit.WithHealthCheck(),              // /healthz and /readyz endpoints
		grpckit.WithGracefulShutdown(30*time.Second),

		// =====================================================================
		// Optional Features (Uncomment to enable)
		// =====================================================================

		// --- Metrics ---
		// Prometheus metrics endpoint at /metrics
		// grpckit.WithMetrics(),

${swagger_line}

		// --- CORS ---
		// Enable Cross-Origin Resource Sharing for browser requests
		// grpckit.WithCORS(),
		//
		// Or with custom configuration:
		// grpckit.WithCORSConfig(grpckit.CORSConfig{
		// 	AllowedOrigins:   []string{"https://example.com"},
		// 	AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		// 	AllowedHeaders:   []string{"Authorization", "Content-Type"},
		// 	AllowCredentials: true,
		// 	MaxAge:           3600,
		// }),

		// --- Authentication ---
		// Uncomment WithAuth and configure public/protected endpoints
		// grpckit.WithAuth(authFunc),
		//
		// Option 1: Define public endpoints (everything else requires auth)
		// grpckit.WithPublicEndpoints(
		// 	"/healthz",
		// 	"/readyz",
		// 	"/metrics",
		// 	"/swagger/*",
		// ),
		//
		// Option 2: Define protected endpoints (everything else is public)
		// grpckit.WithProtectedEndpoints(
		// 	"/api/v1/${SERVICE_NAME}s/*",
		// ),

		// --- Custom Content Types ---
		// Enable form-urlencoded input (HTML forms)
		// grpckit.WithFormURLEncodedSupport(),
		//
		// Enable XML input/output
		// grpckit.WithXMLSupport(),
		//
		// Enable binary data (file downloads)
		// grpckit.WithBinarySupport(),
		//
		// Enable multipart form data (file uploads)
		// grpckit.WithMultipartSupport(),
		//
		// Enable plain text
		// grpckit.WithTextSupport(),

		// --- JSON Options ---
		// Customize JSON serialization
		// grpckit.WithJSONOptions(grpckit.JSONOptions{
		// 	UseProtoNames:   true,  // Use snake_case instead of camelCase
		// 	EmitUnpopulated: true,  // Include fields with zero values
		// 	Indent:          "  ",  // Pretty print
		// }),

		// --- gRPC Interceptors ---
		// Add custom unary interceptor
		// grpckit.WithUnaryInterceptor(func(
		// 	ctx context.Context,
		// 	req interface{},
		// 	info *grpc.UnaryServerInfo,
		// 	handler grpc.UnaryHandler,
		// ) (interface{}, error) {
		// 	log.Printf("[gRPC] %s", info.FullMethod)
		// 	return handler(ctx, req)
		// }),
		//
		// Add interceptor with endpoint exclusions
		// grpckit.WithUnaryInterceptor(myInterceptor,
		// 	grpckit.ExceptEndpoints("/${SERVICE_NAME}.v1.${service_pascal}Service/List${service_pascal}s"),
		// ),
		//
		// Add stream interceptor
		// grpckit.WithStreamInterceptor(func(
		// 	srv interface{},
		// 	ss grpc.ServerStream,
		// 	info *grpc.StreamServerInfo,
		// 	handler grpc.StreamHandler,
		// ) error {
		// 	log.Printf("[gRPC Stream] %s", info.FullMethod)
		// 	return handler(srv, ss)
		// }),

		// --- HTTP Middleware ---
		// Add custom HTTP middleware (applies to all HTTP requests)
		// grpckit.WithHTTPMiddleware(func(next http.Handler) http.Handler {
		// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 		log.Printf("[HTTP] %s %s", r.Method, r.URL.Path)
		// 		next.ServeHTTP(w, r)
		// 	})
		// }),

		// --- Custom HTTP Handlers ---
		// Add custom HTTP endpoint (not exposed via gRPC)
		// grpckit.WithHTTPHandlerFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		// 	w.WriteHeader(http.StatusOK)
		// 	w.Write([]byte(\`{"status": "received"}\`))
		// }),

	); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
EOF

    print_success "main.go generated"
}

# Generate Makefile
generate_makefile() {
    print_info "Generating Makefile..."

    # Build swagger-related Makefile sections if SWAGGER_URL is provided
    local swagger_phony=""
    local swagger_var=""
    local swagger_dep=""
    local swagger_target=""
    local swagger_clean=""
    local swagger_help=""

    if [[ -n "$SWAGGER_URL" ]]; then
        swagger_phony=" swagger"
        swagger_var="
SWAGGER_URL := ${SWAGGER_URL}
SWAGGER_FILE := api/swagger.json"
        swagger_dep=" swagger"
        swagger_target='
# Fetch swagger spec and generate embedding code
swagger:
	@echo "Fetching swagger spec..."
	@mkdir -p api
	@curl -fsSL "$(SWAGGER_URL)" -o $(SWAGGER_FILE) || (echo "Failed to fetch swagger. Check URL or network access." && exit 1)
	@echo "Generating swagger_gen.go..."
	@echo "package main" > swagger_gen.go
	@echo "" >> swagger_gen.go
	@echo "import (" >> swagger_gen.go
	@echo "	_ \"embed\"" >> swagger_gen.go
	@echo "	\"github.com/gyozatech/grpckit\"" >> swagger_gen.go
	@echo ")" >> swagger_gen.go
	@echo "" >> swagger_gen.go
	@echo "//go:embed api/swagger.json" >> swagger_gen.go
	@echo "var _swaggerData []byte" >> swagger_gen.go
	@echo "" >> swagger_gen.go
	@echo "func init() { grpckit.SetSwaggerData(_swaggerData) }" >> swagger_gen.go
	@echo "Done! Swagger will be embedded at build time."
'
        swagger_clean='
	rm -f api/swagger.json swagger_gen.go'
        swagger_help='
	@echo "  swagger       - Fetch swagger spec and generate embedding code"'
    fi

    cat > "${OUTPUT_DIR}/Makefile" << EOF
.PHONY: all proto build run test clean help${swagger_phony}

# Variables
BINARY_NAME := service
PROTO_DIR := proto${swagger_var}

# Default target
all: proto build

# Generate protobuf code
proto:
	@echo "Generating protobuf code..."
	cd \$(PROTO_DIR) && buf generate
	@echo "Done!"

# Update buf dependencies
proto-update:
	@echo "Updating buf dependencies..."
	cd \$(PROTO_DIR) && buf mod update
	@echo "Done!"
${swagger_target}
# Build the service
build:${swagger_dep}
	@echo "Building..."
	go build -o \$(BINARY_NAME) .
	@echo "Done!"

# Run the service
run:
	@echo "Running service..."
	go run .

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f \$(BINARY_NAME)
	rm -f coverage.out coverage.html${swagger_clean}
	@echo "Done!"

# Tidy go modules
tidy:
	@echo "Tidying modules..."
	go mod tidy
	@echo "Done!"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	@echo "Done!"

# Show help
help:
	@echo "Available targets:"
	@echo "  all           - Generate proto and build (default)"
	@echo "  proto         - Generate protobuf code"
	@echo "  proto-update  - Update buf dependencies"${swagger_help}
	@echo "  build         - Build the service binary"
	@echo "  run           - Run the service"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  clean         - Remove build artifacts"
	@echo "  tidy          - Tidy go modules"
	@echo "  deps          - Download dependencies"
	@echo "  help          - Show this help"
EOF

    print_success "Makefile generated"
}

# Generate .gitignore
generate_gitignore() {
    print_info "Generating .gitignore..."

    # Add swagger to gitignore if enabled
    local swagger_ignore=""
    if [[ -n "$SWAGGER_URL" ]]; then
        swagger_ignore="

# Swagger (generated at build time via 'make swagger')
api/swagger.json
swagger_gen.go"
    fi

    cat > "${OUTPUT_DIR}/.gitignore" << EOF
# Binaries
service
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary
*.test

# Coverage
coverage.out
coverage.html

# Go workspace
go.work${swagger_ignore}

# IDE
.idea/
.vscode/
*.swp
*.swo
*~

# OS
.DS_Store
Thumbs.db

# Local env
.env
.env.local
EOF

    print_success ".gitignore generated"
}

# Generate README
generate_readme() {
    local service_pascal=$(to_pascal_case "$SERVICE_NAME")

    print_info "Generating README.md..."

    cat > "${OUTPUT_DIR}/README.md" << EOF
# ${service_pascal} Service

A gRPC + REST microservice built with [grpckit](https://github.com/gyozatech/grpckit).

## Quick Start

### 1. Generate Proto Code

\`\`\`bash
make proto
\`\`\`

### 2. Run the Service

\`\`\`bash
make run
\`\`\`

The service will be available at:
- **gRPC**: localhost:${GRPC_PORT}
- **REST**: localhost:${HTTP_PORT}

## API Endpoints

### REST

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/v1/${SERVICE_NAME}s | Create a ${SERVICE_NAME} |
| GET | /api/v1/${SERVICE_NAME}s | List all ${SERVICE_NAME}s |
| GET | /api/v1/${SERVICE_NAME}s/{id} | Get a ${SERVICE_NAME} by ID |
| PUT | /api/v1/${SERVICE_NAME}s/{id} | Update a ${SERVICE_NAME} |
| DELETE | /api/v1/${SERVICE_NAME}s/{id} | Delete a ${SERVICE_NAME} |

### Health Checks

| Endpoint | Description |
|----------|-------------|
| /healthz | Liveness probe |
| /readyz | Readiness probe |

## Example Requests

### Create a ${SERVICE_NAME}

\`\`\`bash
curl -X POST http://localhost:${HTTP_PORT}/api/v1/${SERVICE_NAME}s \\
  -H "Content-Type: application/json" \\
  -d '{"name": "My ${service_pascal}", "description": "A test ${SERVICE_NAME}"}'
\`\`\`

### List ${SERVICE_NAME}s

\`\`\`bash
curl http://localhost:${HTTP_PORT}/api/v1/${SERVICE_NAME}s
\`\`\`

### gRPC (using grpcurl)

\`\`\`bash
# List services
grpcurl -plaintext localhost:${GRPC_PORT} list

# Create ${SERVICE_NAME}
grpcurl -plaintext -d '{"name": "Test", "description": "A test"}' \\
  localhost:${GRPC_PORT} ${SERVICE_NAME}.v1.${service_pascal}Service/Create${service_pascal}
\`\`\`

## Configuration

Edit \`main.go\` to enable optional features:

- **Metrics**: Uncomment \`grpckit.WithMetrics()\`
- **Swagger UI**: Uncomment \`grpckit.WithSwagger(...)\`
- **CORS**: Uncomment \`grpckit.WithCORS()\`
- **Authentication**: Uncomment \`grpckit.WithAuth(...)\`

See the comments in \`main.go\` for all available options.

## Project Structure

\`\`\`
.
├── main.go                    # Entry point with grpckit configuration
├── internal/
│   └── service/
│       └── ${SERVICE_NAME}_service.go  # Service implementation
├── proto/
│   ├── ${SERVICE_NAME}.proto           # Proto definition
│   ├── buf.yaml               # Buf configuration
│   ├── buf.gen.yaml           # Code generation config
│   └── gen/                   # Generated code
├── Makefile                   # Build commands
└── README.md                  # This file
\`\`\`

## Development

\`\`\`bash
# Generate proto code
make proto

# Run tests
make test

# Build binary
make build

# Run with live reload (using air or similar)
air
\`\`\`

## License

MIT
EOF

    print_success "README.md generated"
}

# Print next steps
print_next_steps() {
    echo ""
    echo "=============================================="
    print_success "Service boilerplate created successfully!"
    echo "=============================================="
    echo ""
    echo -e "Next steps:"
    echo ""
    echo -e "  1. Navigate to the project:"
    echo -e "     ${BLUE}cd ${OUTPUT_DIR}${NC}"
    echo ""
    echo -e "  2. Initialize Go modules:"
    echo -e "     ${BLUE}go mod tidy${NC}"
    echo ""

    if [[ -n "$PROTO_URL" ]]; then
        echo -e "  3. Configure your proto dependency:"
        echo -e "     - The generated code must be available as a Go module"
        echo -e "     - Update go.mod to import the proto package"
        echo -e "     - Proto source: ${BLUE}${PROTO_URL}${NC}"
        echo ""
        echo -e "  4. Implement the service methods:"
        echo -e "     - Edit ${BLUE}internal/service/${SERVICE_NAME}_service.go${NC}"
        echo -e "     - Implement the RPC methods from your proto"
        echo ""
        echo -e "  5. Run the service:"
        echo -e "     ${BLUE}go run .${NC}"
    else
        echo -e "  3. Generate proto code:"
        echo -e "     ${BLUE}make proto${NC}"
        echo ""
        echo -e "  4. Run the service:"
        echo -e "     ${BLUE}make run${NC}"
    fi
    echo ""
    echo -e "  Test the API:"
    echo -e "     ${BLUE}curl http://localhost:${HTTP_PORT}/healthz${NC}"
    echo ""
    echo -e "  Customize options in main.go:"
    echo -e "     - Enable metrics, swagger, CORS, auth, etc."
    echo -e "     - See comments for all available options"
    echo ""
}

# Main execution
main() {
    parse_args "$@"

    echo ""
    echo "=============================================="
    echo "  grpckit Service Generator"
    echo "=============================================="
    echo ""
    print_info "Service name: $SERVICE_NAME"
    print_info "Module path:  $MODULE_PATH"
    print_info "Output dir:   $OUTPUT_DIR"
    print_info "gRPC port:    $GRPC_PORT"
    print_info "HTTP port:    $HTTP_PORT"
    if [[ -n "$PROTO_URL" ]]; then
        print_info "Proto source: $PROTO_URL"
    fi
    if [[ -n "$GRPCKIT_VERSION" ]]; then
        print_info "grpckit ver:  $GRPCKIT_VERSION"
    fi
    if [[ -n "$SWAGGER_URL" ]]; then
        print_info "Swagger:      $SWAGGER_URL"
    fi
    echo ""

    create_directories
    generate_go_mod

    if [[ -n "$PROTO_URL" ]]; then
        # Use external proto file - read it to extract info, don't copy
        fetch_proto_content
        extract_proto_info
        generate_service_stub
        generate_main_for_external_proto
        # No buf config needed - user manages proto generation externally
    else
        # Generate default proto and service
        generate_proto
        generate_service
        generate_main
        generate_buf_yaml
        generate_buf_gen_yaml
    fi

    generate_makefile
    generate_gitignore
    generate_readme

    print_next_steps
}

main "$@"
