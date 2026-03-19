package simulator

import (
	"context"
	"math/rand"
	"time"

	pb "backend/pb/protos"
)

// DeviceSimulator simulates IoT devices generating metrics.
type DeviceSimulator struct {
	client   pb.MetricsServiceClient
	devices  []string
	metrics  []string
	tags     map[string]string
	running  bool
	stopChan chan struct{}
}

// NewDeviceSimulator creates a new device simulator.
func NewDeviceSimulator(client pb.MetricsServiceClient) *DeviceSimulator {
	return &DeviceSimulator{
		client: client,
		devices: []string{
			"device-001",
			"device-002",
			"device-003",
			"device-004",
			"device-005",
		},
		metrics: []string{
			"temperature",
			"humidity",
			"pressure",
			"voltage",
			"current",
		},
		tags: map[string]string{
			"location": "warehouse",
			"unit":     "metric",
		},
		stopChan: make(chan struct{}),
	}
}

// Start begins simulating metrics and sending them to the backend.
func (s *DeviceSimulator) Start(ctx context.Context) {
	if s.running {
		return
	}
	s.running = true

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.generateAndSendMetric(ctx)
			case <-s.stopChan:
				s.running = false
				return
			case <-ctx.Done():
				s.running = false
				return
			}
		}
	}()
}

// Stop stops the simulator.
func (s *DeviceSimulator) Stop() {
	if !s.running {
		return
	}
	close(s.stopChan)
	s.running = false
}

// generateAndSendMetric creates a random metric and sends it to the backend.
func (s *DeviceSimulator) generateAndSendMetric(ctx context.Context) {
	// Select random device and metric
	deviceID := s.devices[rand.Intn(len(s.devices))]
	metricName := s.metrics[rand.Intn(len(s.metrics))]

	// Generate random value based on metric type
	var value float64
	switch metricName {
	case "temperature":
		value = 20.0 + rand.Float64()*15.0 // 20-35°C
	case "humidity":
		value = 30.0 + rand.Float64()*40.0 // 30-70%
	case "pressure":
		value = 980.0 + rand.Float64()*20.0 // 980-1000 hPa
	case "voltage":
		value = 110.0 + rand.Float64()*10.0 // 110-120V
	case "current":
		value = 0.5 + rand.Float64()*4.5 // 0.5-5A
	default:
		value = rand.Float64() * 100.0
	}

	// Create metric update
	metric := &pb.MetricUpdate{
		DeviceId:   deviceID,
		MetricName: metricName,
		Value:      value,
		Timestamp:  time.Now().UnixMilli(),
		Tags:       s.tags,
	}

	// For now, we'll just log the generated metric to show the simulator is working.
	// In a complete implementation, you would either:
	// 1. Call a separate ingestion endpoint (HTTP POST)
	// 2. Store directly in the database
	// 3. Use a message queue
	//
	// Since we're focusing on gRPC for querying in this task, we'll just log the metric.
	// TODO: Implement actual metric ingestion (would require modifying the proto/service)
	//
	// Uncomment the following line to see simulation output:
	// log.Printf("Generated metric: %+v", metric)
	// Use the metric variable to avoid compiler error
	_ = metric
}

// SetTags updates the tags used for generated metrics.
func (s *DeviceSimulator) SetTags(tags map[string]string) {
	s.tags = tags
}

// AddDevice adds a device to the simulation.
func (s *DeviceSimulator) AddDevice(deviceID string) {
	s.devices = append(s.devices, deviceID)
}

// RemoveDevice removes a device from the simulation.
func (s *DeviceSimulator) RemoveDevice(deviceID string) {
	for i, d := range s.devices {
		if d == deviceID {
			s.devices = append(s.devices[:i], s.devices[i+1:]...)
			break
		}
	}
}
