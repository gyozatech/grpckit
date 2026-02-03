package harness

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"

	benchpb "github.com/gyozatech/grpckit/benchmark/proto/gen"
	"google.golang.org/protobuf/proto"
)

// TwirpHarness wraps a Twirp-style service for benchmarking.
// Since we don't have the generated Twirp code, this implements a simplified
// Twirp-compatible HTTP handler that follows Twirp's conventions.
type TwirpHarness struct {
	httpServer *httptest.Server
}

// TwirpConfig contains configuration options for the Twirp harness.
type TwirpConfig struct {
	// ServiceImpl is the service implementation to use.
	ServiceImpl benchpb.BenchmarkServiceServer
}

// twirpHandler handles Twirp-style requests.
type twirpHandler struct {
	impl benchpb.BenchmarkServiceServer
}

// NewTwirpHarness creates a new Twirp-style harness.
func NewTwirpHarness(cfg TwirpConfig) (*TwirpHarness, error) {
	if cfg.ServiceImpl == nil {
		return nil, ErrNilServiceImpl
	}

	handler := &twirpHandler{impl: cfg.ServiceImpl}

	// Create HTTP test server with Twirp-style routing
	mux := http.NewServeMux()

	// Twirp uses POST with /twirp/<package>.<Service>/<Method>
	mux.HandleFunc("/twirp/benchmark.v1.BenchmarkService/Ping", handler.handlePing)
	mux.HandleFunc("/twirp/benchmark.v1.BenchmarkService/Echo", handler.handleEcho)
	mux.HandleFunc("/twirp/benchmark.v1.BenchmarkService/GetItem", handler.handleGetItem)
	mux.HandleFunc("/twirp/benchmark.v1.BenchmarkService/CreateItem", handler.handleCreateItem)
	mux.HandleFunc("/twirp/benchmark.v1.BenchmarkService/ListItems", handler.handleListItems)

	httpServer := httptest.NewServer(mux)

	return &TwirpHarness{
		httpServer: httpServer,
	}, nil
}

// GRPCClient returns nil as Twirp doesn't use gRPC.
func (h *TwirpHarness) GRPCClient() benchpb.BenchmarkServiceClient {
	return nil
}

// HTTPClient returns the HTTP client configured for the test server.
func (h *TwirpHarness) HTTPClient() *http.Client {
	return h.httpServer.Client()
}

// BaseURL returns the base URL for requests.
func (h *TwirpHarness) BaseURL() string {
	return h.httpServer.URL
}

// Close shuts down the harness and releases resources.
func (h *TwirpHarness) Close() {
	if h.httpServer != nil {
		h.httpServer.Close()
	}
}

// Twirp handler implementations.

func (t *twirpHandler) handlePing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	req := &benchpb.PingRequest{}
	if err := t.readRequest(r, req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := t.impl.Ping(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	t.writeResponse(w, r, resp)
}

func (t *twirpHandler) handleEcho(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	req := &benchpb.EchoRequest{}
	if err := t.readRequest(r, req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := t.impl.Echo(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	t.writeResponse(w, r, resp)
}

func (t *twirpHandler) handleGetItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	req := &benchpb.GetItemRequest{}
	if err := t.readRequest(r, req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := t.impl.GetItem(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	t.writeResponse(w, r, resp)
}

func (t *twirpHandler) handleCreateItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	req := &benchpb.CreateItemRequest{}
	if err := t.readRequest(r, req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := t.impl.CreateItem(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	t.writeResponse(w, r, resp)
}

func (t *twirpHandler) handleListItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	req := &benchpb.ListItemsRequest{}
	if err := t.readRequest(r, req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := t.impl.ListItems(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	t.writeResponse(w, r, resp)
}

func (t *twirpHandler) readRequest(r *http.Request, msg proto.Message) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	contentType := r.Header.Get("Content-Type")
	if contentType == "application/protobuf" {
		return proto.Unmarshal(body, msg)
	}
	// Default to JSON
	return json.Unmarshal(body, msg)
}

func (t *twirpHandler) writeResponse(w http.ResponseWriter, r *http.Request, msg proto.Message) {
	accept := r.Header.Get("Content-Type")
	if accept == "application/protobuf" {
		w.Header().Set("Content-Type", "application/protobuf")
		data, err := proto.Marshal(msg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(data)
		return
	}

	// Default to JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(msg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// TwirpClient provides a simple client for the Twirp harness.
type TwirpClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewTwirpClient creates a new Twirp client.
func NewTwirpClient(httpClient *http.Client, baseURL string) *TwirpClient {
	return &TwirpClient{
		httpClient: httpClient,
		baseURL:    baseURL,
	}
}

// Ping calls the Ping method.
func (c *TwirpClient) Ping(ctx context.Context, req *benchpb.PingRequest) (*benchpb.PingResponse, error) {
	resp := &benchpb.PingResponse{}
	err := c.call(ctx, "/twirp/benchmark.v1.BenchmarkService/Ping", req, resp)
	return resp, err
}

// Echo calls the Echo method.
func (c *TwirpClient) Echo(ctx context.Context, req *benchpb.EchoRequest) (*benchpb.EchoResponse, error) {
	resp := &benchpb.EchoResponse{}
	err := c.call(ctx, "/twirp/benchmark.v1.BenchmarkService/Echo", req, resp)
	return resp, err
}

// GetItem calls the GetItem method.
func (c *TwirpClient) GetItem(ctx context.Context, req *benchpb.GetItemRequest) (*benchpb.GetItemResponse, error) {
	resp := &benchpb.GetItemResponse{}
	err := c.call(ctx, "/twirp/benchmark.v1.BenchmarkService/GetItem", req, resp)
	return resp, err
}

// CreateItem calls the CreateItem method.
func (c *TwirpClient) CreateItem(ctx context.Context, req *benchpb.CreateItemRequest) (*benchpb.CreateItemResponse, error) {
	resp := &benchpb.CreateItemResponse{}
	err := c.call(ctx, "/twirp/benchmark.v1.BenchmarkService/CreateItem", req, resp)
	return resp, err
}

// ListItems calls the ListItems method.
func (c *TwirpClient) ListItems(ctx context.Context, req *benchpb.ListItemsRequest) (*benchpb.ListItemsResponse, error) {
	resp := &benchpb.ListItemsResponse{}
	err := c.call(ctx, "/twirp/benchmark.v1.BenchmarkService/ListItems", req, resp)
	return resp, err
}

func (c *TwirpClient) call(ctx context.Context, path string, req, resp proto.Message) error {
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	httpReq.Body = io.NopCloser(io.NopCloser(nil))
	httpReq.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(io.NopCloser(nil)), nil
	}

	// Re-create with body
	httpReq, err = http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, io.NopCloser(
		&bytesReader{data: body},
	))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()

	return json.NewDecoder(httpResp.Body).Decode(resp)
}

// bytesReader implements io.Reader for a byte slice.
type bytesReader struct {
	data []byte
	pos  int
}

func (r *bytesReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
