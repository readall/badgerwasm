package main

import (
	"context"
	"log"
	"strconv"
	"syscall/js"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "frontend/backend/pb"
)

// Exported function to update the UI from JavaScript
func updateUI(this js.Value, args []js.Value) interface{} {
	if len(args) != 1 {
		return nil
	}
	message := js.Global().Get("document").Call("getElementById", "output")
	message.Set("innerText", args[0].String())
	return nil
}

func registerCallbacks() {
	js.Global().Set("updateUI", js.FuncOf(updateUI))
}

func main() {
	registerCallbacks()

	// Start the gRPC client in a goroutine
	go startGRPCClient()

	// Make sure the Go WASM program doesn't exit
	c := make(chan struct{}, 0)
	<-c
}

func startGRPCClient() {
	// Set up connection to the gRPC server
	conn, err := grpc.DialContext(
		context.Background(),
		"localhost:50051", // Backend gRPC server address
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Printf("Failed to connect: %v", err)
		updateUI(js.Null(), []js.Value{js.ValueOf("Connection failed: " + err.Error())})
		return
	}
	defer conn.Close()

	// Create a gRPC client
	client := pb.NewMetricsServiceClient(conn)

	// Create a subscription request
	req := &pb.SubscriptionRequest{
		DeviceIds:   []string{"device1", "device2"},
		MetricTypes: []string{"temperature", "humidity"},
	}

	// Subscribe to metric updates
	stream, err := client.SubscribeMetrics(context.Background(), req)
	if err != nil {
		log.Printf("Failed to subscribe: %v", err)
		updateUI(js.Null(), []js.Value{js.ValueOf("Subscription failed: " + err.Error())})
		return
	}

	// Listen for updates
	for {
		resp, err := stream.Recv()
		if err != nil {
			log.Printf("Error receiving stream: %v", err)
			updateUI(js.Null(), []js.Value{js.ValueOf("Stream error: " + err.Error())})
			break
		}
		message := "Device: " + resp.GetDeviceId() +
			", Metric: " + resp.GetMetricName() +
			", Value: " + strconv.FormatFloat(resp.GetValue(), 'f', -1, 64) +
			", Time: " + time.UnixMilli(resp.GetTimestamp()).Format("15:04:05")
		updateUI(js.Null(), []js.Value{js.ValueOf(message)})
		time.Sleep(100 * time.Millisecond) // Prevent busy loop
	}
}
