package main

import (
	"context"
	"log"
	"sync"
	"time"

	// import the library
	"github.com/gyozatech/grpckit"
	// import the proto
	pb "github.com/gyozatech/grpckit/example/proto/gen"
)

// ItemService implements the ItemServiceServer interface.
// This is where you implement your business logic.
type ItemService struct {
	pb.UnimplementedItemServiceServer
	// the following simulates an in memory database
	mu    sync.RWMutex
	items map[string]*pb.Item
}

// NewItemService creates a new ItemService.
func NewItemService() *ItemService {
	return &ItemService{
		items: make(map[string]*pb.Item),
	}
}

// CreateItem creates a new item.
func (s *ItemService) CreateItem(ctx context.Context, req *pb.CreateItemRequest) (*pb.CreateItemResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Unix()
	item := &pb.Item{
		Id:          generateID(),
		Name:        req.Name,
		Description: req.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	s.items[item.Id] = item
	log.Printf("Created item: %s", item.Id)

	return &pb.CreateItemResponse{Item: item}, nil
}

// GetItem retrieves an item by ID.
func (s *ItemService) GetItem(ctx context.Context, req *pb.GetItemRequest) (*pb.GetItemResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.items[req.Id]
	if !ok {
		return nil, grpckit.ErrNotFound
	}

	return &pb.GetItemResponse{Item: item}, nil
}

// ListItems retrieves a list of items.
func (s *ItemService) ListItems(ctx context.Context, req *pb.ListItemsRequest) (*pb.ListItemsResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]*pb.Item, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}

	return &pb.ListItemsResponse{Items: items}, nil
}

// UpdateItem updates an existing item.
func (s *ItemService) UpdateItem(ctx context.Context, req *pb.UpdateItemRequest) (*pb.UpdateItemResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.items[req.Id]
	if !ok {
		return nil, grpckit.ErrNotFound
	}

	item.Name = req.Name
	item.Description = req.Description
	item.UpdatedAt = time.Now().Unix()

	return &pb.UpdateItemResponse{Item: item}, nil
}

// DeleteItem deletes an item by ID.
func (s *ItemService) DeleteItem(ctx context.Context, req *pb.DeleteItemRequest) (*pb.DeleteItemResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.items[req.Id]; !ok {
		return nil, grpckit.ErrNotFound
	}

	delete(s.items, req.Id)
	log.Printf("Deleted item: %s", req.Id)

	return &pb.DeleteItemResponse{Success: true}, nil
}

// generateID generates a simple unique ID.
func generateID() string {
	return time.Now().Format("20060102150405.000000")
}
