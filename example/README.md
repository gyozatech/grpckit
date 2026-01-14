# grpckit Example

A complete example showing how to build a CRUD microservice with grpckit.

## Structure

```
example/
├── main.go              # Entry point - grpckit configuration
├── item_service.go      # Service implementation (business logic)
└── proto/
    ├── item.proto       # Proto definition with REST annotations
    ├── buf.yaml         # Buf configuration
    ├── buf.gen.yaml     # Code generation config
    └── gen/             # Generated code
        ├── item.pb.go
        ├── item_grpc.pb.go
        ├── item.pb.gw.go
        └── item.swagger.json
```

## Running the Example

### 1. Generate Proto Code (if needed)

```bash
cd proto
buf generate
```

### 2. Run the Server

```bash
go run .
```

Output:
```
Starting grpckit example server...
gRPC server listening on :9090
HTTP server listening on :8080
```

## Testing the API

> **Note**: The example configures `/api/v1/items` and `/api/v1/items/*` as public endpoints for demo purposes. If you remove these from `WithPublicEndpoints()`, you'll need to include the `Authorization: Bearer <token>` header.

### REST Endpoints (curl)

**Create an item (public endpoint):**
```bash
curl -X POST http://localhost:8080/api/v1/items \
  -H "Content-Type: application/json" \
  -d '{"name": "My Item", "description": "A test item"}'
```

**Create an item (with auth token):**
```bash
curl -X POST http://localhost:8080/api/v1/items \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer my-secret-token" \
  -d '{"name": "My Item", "description": "A test item"}'
```

**List items:**
```bash
curl http://localhost:8080/api/v1/items
```

**Get an item:**
```bash
curl http://localhost:8080/api/v1/items/{id}
```

**Update an item:**
```bash
curl -X PUT http://localhost:8080/api/v1/items/{id} \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer my-secret-token" \
  -d '{"name": "Updated Name", "description": "Updated description"}'
```

**Delete an item:**
```bash
curl -X DELETE http://localhost:8080/api/v1/items/{id} \
  -H "Authorization: Bearer my-secret-token"
```

**Patch an item (form-urlencoded input, XML output):**

This endpoint demonstrates custom content types. Send form data, receive XML:

```bash
# First create an item to get an ID
ITEM_ID=$(curl -s -X POST http://localhost:8080/api/v1/items \
  -H "Content-Type: application/json" \
  -d '{"name": "Original Name", "description": "Original description"}' \
  | grep -o '"id":"[^"]*"' | cut -d'"' -f4)

echo "Created item: $ITEM_ID"

# Patch using form-urlencoded input, request XML output
curl -X PATCH "http://localhost:8080/api/v1/items/${ITEM_ID}" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "Accept: application/xml" \
  -d "name=Patched+Name&description=Updated+via+form"
```

Response (XML):
```xml
<PatchItemResponse>
  <item>
    <id>20240115120000.000000</id>
    <name>Patched Name</name>
    <description>Updated via form</description>
    <created_at>1705320000</created_at>
    <updated_at>1705320060</updated_at>
  </item>
</PatchItemResponse>
```

### Health & Metrics

**Health check:**
```bash
curl http://localhost:8080/healthz
# {"status":"ok"}

curl http://localhost:8080/readyz
# {"status":"ok"}
```

**Prometheus metrics:**
```bash
curl http://localhost:8080/metrics
```

### Swagger UI

Open in browser: http://localhost:8080/swagger/

### gRPC (grpcurl)

Using [grpcurl](https://github.com/fullstorydev/grpcurl):

```bash
# List services (no auth required for reflection)
grpcurl -plaintext localhost:9090 list

# Describe a service
grpcurl -plaintext localhost:9090 describe item.v1.ItemService
```

**Without authentication (if endpoint is public):**
```bash
# Create item
grpcurl -plaintext -d '{"name": "Test", "description": "A test"}' \
  localhost:9090 item.v1.ItemService/CreateItem

# List items
grpcurl -plaintext localhost:9090 item.v1.ItemService/ListItems

# Get item
grpcurl -plaintext -d '{"id": "your-item-id"}' \
  localhost:9090 item.v1.ItemService/GetItem
```

**With Bearer token authentication:**
```bash
# Create item with auth
grpcurl -plaintext \
  -H "Authorization: Bearer my-secret-token" \
  -d '{"name": "Test", "description": "A test"}' \
  localhost:9090 item.v1.ItemService/CreateItem

# List items with auth
grpcurl -plaintext \
  -H "Authorization: Bearer my-secret-token" \
  localhost:9090 item.v1.ItemService/ListItems

# Update item with auth
grpcurl -plaintext \
  -H "Authorization: Bearer my-secret-token" \
  -d '{"id": "your-item-id", "name": "Updated", "description": "Updated desc"}' \
  localhost:9090 item.v1.ItemService/UpdateItem

# Delete item with auth
grpcurl -plaintext \
  -H "Authorization: Bearer my-secret-token" \
  -d '{"id": "your-item-id"}' \
  localhost:9090 item.v1.ItemService/DeleteItem
```

## Code Walkthrough

### Proto Definition (`proto/item.proto`)

```protobuf
service ItemService {
  rpc CreateItem(CreateItemRequest) returns (CreateItemResponse) {
    option (google.api.http) = {
      post: "/api/v1/items"
      body: "*"
    };
  }

  // PatchItem demonstrates custom content types
  // Works with form-urlencoded input and XML output
  rpc PatchItem(PatchItemRequest) returns (PatchItemResponse) {
    option (google.api.http) = {
      patch: "/api/v1/items/{id}"
      body: "*"
    };
  }
  // ... other methods
}
```

The `google.api.http` annotations define REST endpoints that grpc-gateway generates.
The content type is determined by HTTP headers, not the proto definition.

### Service Implementation (`item_service.go`)

```go
type ItemService struct {
    pb.UnimplementedItemServiceServer
    mu    sync.RWMutex
    items map[string]*pb.Item
}

func (s *ItemService) CreateItem(ctx context.Context, req *pb.CreateItemRequest) (*pb.CreateItemResponse, error) {
    // Your business logic here
}
```

Implement the interface generated from your proto. Use `grpckit.ErrNotFound`, `grpckit.ErrUnauthorized`, etc. for common errors.

### Main Entry Point (`main.go`)

```go
func main() {
    grpckit.Run(
        // Register services
        grpckit.WithGRPCService(func(s grpc.ServiceRegistrar) {
            pb.RegisterItemServiceServer(s, NewItemService())
        }),
        grpckit.WithRESTService(pb.RegisterItemServiceHandlerFromEndpoint),

        // Custom content types (for PatchItem endpoint)
        grpckit.WithFormURLEncodedSupport(),  // Accept form data
        grpckit.WithXMLSupport(),              // Return XML responses

        // Enable features
        grpckit.WithHealthCheck(),
        grpckit.WithMetrics(),
        grpckit.WithSwagger("./proto/gen/item.swagger.json"),
    )
}
```

## Adding Authentication

The example includes an auth function (currently allowing public access for demo):

```go
authFunc := func(ctx context.Context, token string) (context.Context, error) {
    if token == "" {
        return nil, grpckit.ErrUnauthorized
    }
    // Validate token and return enriched context
    return context.WithValue(ctx, "user_id", "user-123"), nil
}

grpckit.Run(
    grpckit.WithAuth(authFunc),
    grpckit.WithPublicEndpoints("/healthz", "/readyz", "/metrics"),
    // ...
)
```

To test with auth:
```bash
curl -H "Authorization: Bearer your-token" http://localhost:8080/api/v1/items
```

## Custom Content Types

This example demonstrates grpckit's support for custom content types using the `PatchItem` endpoint.

### Enabling Custom Content Types

In `main.go`, enable the marshalers you need:

```go
grpckit.Run(
    grpckit.WithGRPCService(...),
    grpckit.WithRESTService(...),

    // Enable form-urlencoded input
    grpckit.WithFormURLEncodedSupport(),

    // Enable XML output
    grpckit.WithXMLSupport(),
)
```

### How It Works

The content type is determined by HTTP headers:

| Header | Purpose | Example |
|--------|---------|---------|
| `Content-Type` | Format of request body | `application/x-www-form-urlencoded` |
| `Accept` | Desired response format | `application/xml` |

### PatchItem Endpoint Demo

The `PatchItem` endpoint (`PATCH /api/v1/items/{id}`) demonstrates:
- **Input**: `application/x-www-form-urlencoded` (HTML form data)
- **Output**: `application/xml` (when `Accept: application/xml` header is set)

**Send form data:**
```bash
curl -X PATCH "http://localhost:8080/api/v1/items/123" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "Accept: application/xml" \
  -d "name=New+Name&description=Updated+description"
```

**Form field mapping:**
- Fields map to proto field names (snake_case)
- Nested fields: `address.street=123`
- Repeated fields: `tags=a&tags=b`

### Available Content Types

| Option | Content-Type | Description |
|--------|--------------|-------------|
| `WithFormURLEncodedSupport()` | `application/x-www-form-urlencoded` | HTML form submissions |
| `WithXMLSupport()` | `application/xml` | XML input/output |
| `WithBinarySupport()` | `application/octet-stream` | Raw binary data |
| `WithMultipartSupport()` | `multipart/form-data` | File uploads |
| `WithTextSupport()` | `text/plain` | Plain text |

### Custom JSON Options

Configure JSON behavior:

```go
grpckit.WithJSONOptions(grpckit.JSONOptions{
    UseProtoNames:   true,  // snake_case instead of camelCase
    EmitUnpopulated: true,  // Include zero-value fields
    Indent:          "  ",  // Pretty print
})
```

## Custom HTTP Endpoints (Webhook Example)

This example includes a webhook endpoint that demonstrates custom HTTP handlers outside of proto/gRPC.

### Why Custom HTTP Endpoints?

Some endpoints don't fit the proto model:
- Webhooks from external services (GitHub, Stripe, etc.)
- File uploads/downloads
- Legacy endpoints with specific format requirements
- Endpoints needing different authentication (e.g., signature validation)

### Webhook Implementation

The `/webhook` endpoint in this example:
- Is **NOT** in the proto file (pure HTTP, no gRPC exposure)
- Has its own **dedicated middleware** for signature validation
- Accepts **any payload format** (not constrained by proto)

**Registration in main.go:**

```go
grpckit.WithHTTPHandler("/webhook",
    webhookAuthMiddleware("my-webhook-secret")(
        http.HandlerFunc(webhookHandler),
    ),
),
```

### Testing the Webhook

**Without signature (rejected):**
```bash
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d '{"event": "test"}'
# Response: 401 Missing webhook signature
```

**With invalid signature (rejected):**
```bash
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -H "X-Webhook-Signature: wrong" \
  -d '{"event": "test"}'
# Response: 403 Invalid webhook signature
```

**With valid signature (accepted):**
```bash
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -H "X-Webhook-Signature: valid-signature" \
  -d '{"event": "push", "repo": "myrepo"}'
# Response: {"status": "received", "message": "Webhook processed successfully"}
```

### Key Differences from Proto Endpoints

| Aspect | Proto Endpoints | Custom HTTP Endpoints |
|--------|-----------------|----------------------|
| Definition | In `.proto` file | In Go code |
| gRPC access | Yes | No (HTTP only) |
| Input/Output | Constrained by proto | Any format |
| Middleware | Global only | Per-handler possible |
| Code generation | Automatic | Manual |

## Custom gRPC Interceptors

This example includes custom gRPC interceptors that apply to ALL gRPC calls.

### Interceptors in This Example

**Logging Interceptor** (`loggingUnaryInterceptor`):
- Logs when each gRPC method starts and completes
- Logs errors if they occur

**Timing Interceptor** (`timingUnaryInterceptor`):
- Measures and logs the duration of each gRPC call

**Stream Logging Interceptor** (`loggingStreamInterceptor`):
- Logs streaming RPC calls (start/complete/error)

### Registration

```go
grpckit.Run(
    // ... services ...

    // Logging interceptor - applies to ALL endpoints
    grpckit.WithUnaryInterceptor(loggingUnaryInterceptor),

    // Timing interceptor - skip for ListItems (high-frequency, low-value timing)
    grpckit.WithUnaryInterceptor(timingUnaryInterceptor,
        grpckit.ExceptEndpoints("/item.v1.ItemService/ListItems"),
    ),

    grpckit.WithStreamInterceptor(loggingStreamInterceptor),
)
```

### Excluding Endpoints

Use `ExceptEndpoints` to skip specific endpoints from an interceptor:

```go
grpckit.WithUnaryInterceptor(timingInterceptor,
    grpckit.ExceptEndpoints(
        "/item.v1.ItemService/ListItems",
        "/item.v1.ItemService/HealthCheck",
    ),
)
```

This is useful for:
- High-frequency endpoints where timing overhead matters
- Health check endpoints that shouldn't be logged/timed
- Any endpoint that needs different interceptor behavior

### Testing Interceptors

When you make a gRPC call, you'll see interceptor logs:

```bash
# CreateItem - shows both logging and timing
grpcurl -plaintext -d '{"name": "Test", "description": "A test"}' \
  localhost:9090 item.v1.ItemService/CreateItem
```

Server logs:
```
[gRPC] Start: /item.v1.ItemService/CreateItem
[gRPC] Done: /item.v1.ItemService/CreateItem
[gRPC Timing] /item.v1.ItemService/CreateItem took 1.234ms
```

```bash
# ListItems - timing is excluded, only logging shows
grpcurl -plaintext localhost:9090 item.v1.ItemService/ListItems
```

Server logs (no timing):
```
[gRPC] Start: /item.v1.ItemService/ListItems
[gRPC] Done: /item.v1.ItemService/ListItems
```

### Interceptor Execution Order

```
gRPC Request
  ↓
auth interceptor (built-in)
  ↓
loggingUnaryInterceptor (first registered)
  ↓
timingUnaryInterceptor (second registered)
  ↓
Handler
  ↓
Response flows back through interceptors
```

### HTTP vs gRPC Middleware

| Aspect | HTTP Middleware | gRPC Interceptors |
|--------|----------------|-------------------|
| Applies to | HTTP/REST requests | gRPC calls |
| Registration | `WithHTTPMiddleware()` | `WithUnaryInterceptor()` / `WithStreamInterceptor()` |
| Type | `func(http.Handler) http.Handler` | `grpc.UnaryServerInterceptor` / `grpc.StreamServerInterceptor` |
| Access | Request/Response | Context, Request, Info |

## Customizing

1. **Add new methods**: Update `item.proto`, regenerate with `buf generate`, implement in `item_service.go`
2. **Change ports**: Use `grpckit.WithGRPCPort()` and `grpckit.WithHTTPPort()`
3. **Add more services**: Register multiple services with `WithGRPCService()` and `WithRESTService()`
4. **Add content types**: Use `WithFormURLEncodedSupport()`, `WithXMLSupport()`, etc.
5. **Add custom HTTP endpoints**: Use `WithHTTPHandler()` or `WithHTTPHandlerFunc()`
6. **Add gRPC interceptors**: Use `WithUnaryInterceptor()` or `WithStreamInterceptor()`
