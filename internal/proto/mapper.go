package proto

import (
	"fmt"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/models"
)

// FromModel converts a models.Metrics into a proto Metric.
// It returns an error for unknown metric types.
func FromModel(m models.Metrics) (*Metric, error) {
	pm := &Metric{Id: m.ID}
	switch m.MType {
	case models.Gauge:
		pm.Type = Metric_GAUGE
		if m.Value != nil {
			pm.Value = *m.Value
		}
	case models.Counter:
		pm.Type = Metric_COUNTER
		if m.Delta != nil {
			pm.Delta = *m.Delta
		}
	default:
		return nil, fmt.Errorf("unknown metric type: %q", m.MType)
	}
	return pm, nil
}

// ToModel converts a proto Metric into a models.Metrics.
// It returns an error for unknown metric types.
func ToModel(m *Metric) (models.Metrics, error) {
	metric := models.Metrics{ID: m.GetId()}
	switch m.GetType() {
	case Metric_GAUGE:
		metric.MType = models.Gauge
		value := m.GetValue()
		metric.Value = &value
	case Metric_COUNTER:
		metric.MType = models.Counter
		delta := m.GetDelta()
		metric.Delta = &delta
	default:
		return models.Metrics{}, fmt.Errorf("unknown metric type: %v", m.GetType())
	}
	return metric, nil
}
