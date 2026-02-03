// Package service provides the shared benchmark service implementation.
package service

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	benchpb "github.com/gyozatech/grpckit/benchmark/proto/gen"
)

// Impl implements the BenchmarkService interface.
// It can be used with gRPC, gRPC-gateway, Connect, and Twirp.
type Impl struct {
	benchpb.UnimplementedBenchmarkServiceServer
	counter atomic.Int64
}

// New creates a new service implementation.
func New() *Impl {
	return &Impl{}
}

// Ping returns a simple response with the current timestamp.
func (s *Impl) Ping(ctx context.Context, req *benchpb.PingRequest) (*benchpb.PingResponse, error) {
	return &benchpb.PingResponse{
		Message:   "pong",
		Timestamp: time.Now().UnixNano(),
	}, nil
}

// Echo returns the input message back to the caller.
func (s *Impl) Echo(ctx context.Context, req *benchpb.EchoRequest) (*benchpb.EchoResponse, error) {
	return &benchpb.EchoResponse{
		Message:  req.Message,
		Data:     req.Data,
		Metadata: req.Metadata,
	}, nil
}

// GetItem retrieves an item by ID.
func (s *Impl) GetItem(ctx context.Context, req *benchpb.GetItemRequest) (*benchpb.GetItemResponse, error) {
	return &benchpb.GetItemResponse{
		Item: &benchpb.Item{
			Id:          req.Id,
			Name:        fmt.Sprintf("Item %s", req.Id),
			Description: "A sample item for benchmarking",
			Price:       99.99,
			Tags:        []string{"benchmark", "test"},
			CreatedAt:   time.Now().UnixNano(),
		},
	}, nil
}

// CreateItem creates a new item and returns it with a generated ID.
func (s *Impl) CreateItem(ctx context.Context, req *benchpb.CreateItemRequest) (*benchpb.CreateItemResponse, error) {
	id := s.counter.Add(1)
	return &benchpb.CreateItemResponse{
		Item: &benchpb.Item{
			Id:          fmt.Sprintf("%d", id),
			Name:        req.Name,
			Description: req.Description,
			Price:       req.Price,
			Tags:        req.Tags,
			CreatedAt:   time.Now().UnixNano(),
		},
	}, nil
}

// ListItems returns a list of sample items.
func (s *Impl) ListItems(ctx context.Context, req *benchpb.ListItemsRequest) (*benchpb.ListItemsResponse, error) {
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	items := make([]*benchpb.Item, pageSize)
	for i := int32(0); i < pageSize; i++ {
		items[i] = &benchpb.Item{
			Id:          fmt.Sprintf("item-%d", i+1),
			Name:        fmt.Sprintf("Item %d", i+1),
			Description: "A sample item for benchmarking",
			Price:       float64(i+1) * 10.0,
			Tags:        []string{"benchmark", "test"},
			CreatedAt:   time.Now().UnixNano(),
		}
	}

	return &benchpb.ListItemsResponse{
		Items:         items,
		NextPageToken: "",
	}, nil
}
