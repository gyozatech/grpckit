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

### REST Endpoints

**Create an item:**
```bash
curl -X POST http://localhost:8080/api/v1/items \
  -H "Content-Type: application/json" \
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
  -d '{"name": "Updated Name", "description": "Updated description"}'
```

**Delete an item:**
```bash
curl -X DELETE http://localhost:8080/api/v1/items/{id}
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

### gRPC

Using [grpcurl](https://github.com/fullstorydev/grpcurl):

```bash
# List services
grpcurl -plaintext localhost:9090 list

# Create item
grpcurl -plaintext -d '{"name": "Test", "description": "A test"}' \
  localhost:9090 item.v1.ItemService/CreateItem

# List items
grpcurl -plaintext localhost:9090 item.v1.ItemService/ListItems
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
  // ... other methods
}
```

The `google.api.http` annotations define REST endpoints that grpc-gateway generates.

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

## Customizing

1. **Add new methods**: Update `item.proto`, regenerate with `buf generate`, implement in `item_service.go`
2. **Change ports**: Use `grpckit.WithGRPCPort()` and `grpckit.WithHTTPPort()`
3. **Add more services**: Register multiple services with `WithGRPCService()` and `WithRESTService()`
