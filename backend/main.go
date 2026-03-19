package main

import (
	"log"
	"net"

	"github.com/dgraph-io/badger/v4"
	"google.golang.org/grpc"

	"backend/api"
	pb "backend/pb/protos"
	"backend/storage"
)

func main() {
	// Initialize BadgerDB
	opts := badger.DefaultOptions("./badger-db")
	opts.Logger = nil // Disable built-in logger for cleaner output
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatalf("Failed to open BadgerDB: %v", err)
	}
	defer db.Close()

	// Initialize storage layer
	store := storage.NewBadgerStore(db)

	// Initialize API handler
	handler := api.NewMetricsHandler(store)

	// Create a gRPC server
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterMetricsServiceServer(s, handler)
	log.Println("gRPC server listening on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
