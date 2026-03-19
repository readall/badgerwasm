package main

import (
	"log"
	"net"
	"net/http"

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

	// Create HTTP server to serve frontend files and health endpoint
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Serve index.html for root path
		if r.URL.Path == "/" {
			http.ServeFile(w, r, "../frontend/index.html")
			return
		}
		// Serve static files from frontend directory
		http.FileServer(http.Dir("../frontend")).ServeHTTP(w, r)
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Start HTTP server in a goroutine
	go func() {
		log.Println("HTTP server listening on :8080")
		if err := http.ListenAndServe(":8080", mux); err != nil {
			log.Fatalf("Failed to serve HTTP: %v", err)
		}
	}()

	// Wait for gRPC server to finish
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}
