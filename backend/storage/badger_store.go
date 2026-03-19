package storage

import (
	"context"
	"encoding/json"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/google/uuid"
)

// BadgerStore handles all interactions with BadgerDB.
type BadgerStore struct {
	db *badger.DB
}

// NewBadgerStore creates a new BadgerStore instance.
func NewBadgerStore(db *badger.DB) *BadgerStore {
	return &BadgerStore{db: db}
}

// Metric represents a metric point stored in the database.
type Metric struct {
	ID         string            `json:"id"`
	DeviceID   string            `json:"device_id"`
	MetricName string            `json:"metric_name"`
	Value      float64           `json:"value"`
	Timestamp  int64             `json:"timestamp"` // Unix timestamp in milliseconds
	Tags       map[string]string `json:"tags"`
	CreatedAt  time.Time         `json:"created_at"`
}

// SaveMetric saves a metric to the database.
func (s *BadgerStore) SaveMetric(ctx context.Context, metric *Metric) error {
	// Generate ID if not present
	if metric.ID == "" {
		metric.ID = uuid.New().String()
	}

	// Set creation time if not set
	if metric.CreatedAt.IsZero() {
		metric.CreatedAt = time.Now()
	}

	// Serialize metric to JSON
	data, err := json.Marshal(metric)
	if err != nil {
		return err
	}

	// Key format: metric:{deviceID}:{metricName}:{timestamp}
	key := []byte("metric:" + metric.DeviceID + ":" + metric.MetricName + ":" + string(metric.Timestamp))

	// Save to BadgerDB
	err = s.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, data)
	})

	return err
}

// GetMetrics retrieves metrics for a device and metric name within a time range.
func (s *BadgerStore) GetMetrics(ctx context.Context, deviceID, metricName string, startTime, endTime int64) ([]*Metric, error) {
	var metrics []*Metric

	// Key prefix format: metric:{deviceID}:{metricName}:
	prefix := []byte("metric:" + deviceID + ":" + metricName + ":")

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			var m Metric

			err := item.Value(func(val []byte) error {
				return json.Unmarshal(val, &m)
			})
			if err != nil {
				return err
			}

			// Filter by time range (already partially filtered by prefix, but double-check)
			if m.Timestamp >= startTime && m.Timestamp <= endTime {
				metrics = append(metrics, &m)
			}
		}

		return nil
	})

	return metrics, err
}

// GetLatestMetric retrieves the most recent metric for a device and metric name.
func (s *BadgerStore) GetLatestMetric(ctx context.Context, deviceID, metricName string) (*Metric, error) {
	var latest *Metric

	// Key prefix format: metric:{deviceID}:{metricName}:
	prefix := []byte("metric:" + deviceID + ":" + metricName + ":")

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		it := txn.NewIterator(opts)
		defer it.Close()

		// Seek to the end of the prefix range to get the latest
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			var m Metric

			err := item.Value(func(val []byte) error {
				return json.Unmarshal(val, &m)
			})
			if err != nil {
				return err
			}

			if latest == nil || m.Timestamp > latest.Timestamp {
				latest = &m
			}
		}

		return nil
	})

	return latest, err
}

// GetDevices returns a list of all unique device IDs in the database.
func (s *BadgerStore) GetDevices(ctx context.Context) ([]string, error) {
	deviceIDs := make(map[string]bool)

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false // We only need keys for this operation
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek([]byte("metric:")); it.ValidForPrefix([]byte("metric:")); it.Next() {
			item := it.Item()
			key := item.Key()

			// Parse key format: metric:{deviceID}:{metricName}:{timestamp}
			// Extract deviceID (second segment)
			parts := string(key)
			if len(parts) > 0 {
				// Split by colon and get deviceID (index 1)
				var deviceID string
				var count int
				for i, ch := range parts {
					if ch == ':' {
						count++
						if count == 1 { // First colon after "metric"
							deviceIDStart := i + 1
							// Find second colon
							for j := i + 1; j < len(parts); j++ {
								if parts[j] == ':' {
									deviceID = parts[deviceIDStart:j]
									deviceIDs[deviceID] = true
									break
								}
							}
							break
						}
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Convert map to slice
	devices := make([]string, 0, len(deviceIDs))
	for deviceID := range deviceIDs {
		devices = append(devices, deviceID)
	}

	return devices, nil
}
