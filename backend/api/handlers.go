package api

import (
	"context"
	"log"

	pb "backend/pb/protos"
	"backend/storage"
)

// MetricsHandler implements the gRPC service for metrics.
type MetricsHandler struct {
	pb.UnimplementedMetricsServiceServer
	store *storage.BadgerStore
}

// NewMetricsHandler creates a new metrics handler.
func NewMetricsHandler(store *storage.BadgerStore) *MetricsHandler {
	return &MetricsHandler{store: store}
}

// SubscribeMetrics streams real-time metric updates.
func (h *MetricsHandler) SubscribeMetrics(req *pb.SubscriptionRequest, stream pb.MetricsService_SubscribeMetricsServer) error {
	// For now, we'll return an error indicating this is not implemented
	// In a full implementation, we would set up a mechanism to stream updates
	// when new metrics arrive (e.g., using channels or a message broker)
	log.Printf("SubscribeMetrics called for devices: %v, metrics: %v", req.DeviceIds, req.MetricTypes)
	return nil
}

// GetHistoricalMetrics streams historical metric points.
func (h *MetricsHandler) GetHistoricalMetrics(req *pb.HistoricalRequest, stream pb.MetricsService_GetHistoricalMetricsServer) error {
	log.Printf("GetHistoricalMetrics called for device: %s, metric: %s, from: %d, to: %d", req.DeviceId, req.MetricName, req.StartTime, req.EndTime)

	// Retrieve metrics from storage
	metrics, err := h.store.GetMetrics(context.Background(), req.DeviceId, req.MetricName, req.StartTime, req.EndTime)
	if err != nil {
		return err
	}

	// Stream each metric point
	for _, metric := range metrics {
		// Convert internal Metric to protobuf MetricPoint
		point := &pb.MetricPoint{
			Value:     metric.Value,
			Timestamp: metric.Timestamp,
		}

		if err := stream.Send(point); err != nil {
			return err
		}
	}

	return nil
}

// GetDevices returns the list of available devices.
func (h *MetricsHandler) GetDevices(ctx context.Context, req *pb.DeviceRequest) (*pb.DeviceList, error) {
	log.Printf("GetDevices called")

	// Get devices from storage
	deviceIDs, err := h.store.GetDevices(context.Background())
	if err != nil {
		return nil, err
	}

	return &pb.DeviceList{DeviceIds: deviceIDs}, nil
}

// UpdateConfig updates the configuration.
func (h *MetricsHandler) UpdateConfig(ctx context.Context, req *pb.ConfigUpdate) (*pb.ConfigResponse, error) {
	log.Printf("UpdateConfig called for dashboard: %s, settings: %v", req.DashboardId, req.Settings)

	// In a real implementation, we would store the configuration
	// For now, we'll just return success
	return &pb.ConfigResponse{Success: true, Message: "Configuration updated"}, nil
}
