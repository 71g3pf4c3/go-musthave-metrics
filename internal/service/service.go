package service

import (
	"context"
	"errors"
	"strconv"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/models"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/repository"
)

var ErrBadRequest = errors.New("bad request")

type Service interface {
	Dump(ctx context.Context, path string) error
	Restore(ctx context.Context, path string) error
	Ping(ctx context.Context) error
	List(ctx context.Context) (map[string]float64, map[string]int64, error)
	GetValue(ctx context.Context, kind string, name string) (string, error)
	Update(ctx context.Context, kind string, name string, value string) error
	JSONUpdate(ctx context.Context, metric models.Metrics) error
	JSONGet(ctx context.Context, metric models.Metrics) (models.Metrics, error)
	BatchUpdate(ctx context.Context, metrics []models.Metrics) error
}

type MetricsService struct {
	repo repository.Repository
}

func New(repo repository.Repository) *MetricsService {
	return &MetricsService{repo: repo}
}

func (s *MetricsService) Dump(ctx context.Context, path string) error {
	return s.repo.Dump(ctx, path)
}

func (s *MetricsService) Restore(ctx context.Context, path string) error {
	return s.repo.Restore(ctx, path)
}

func (s *MetricsService) Ping(ctx context.Context) error {
	return s.repo.Ping(ctx)
}

func (s *MetricsService) List(ctx context.Context) (map[string]float64, map[string]int64, error) {
	gauges, err := s.repo.GetAllGauge(ctx)
	if err != nil {
		return nil, nil, err
	}
	counters, err := s.repo.GetAllCounter(ctx)
	if err != nil {
		return nil, nil, err
	}
	return gauges, counters, nil
}

func (s *MetricsService) GetValue(ctx context.Context, kind string, name string) (string, error) {
	return s.repo.GetValue(ctx, name, kind)
}

func (s *MetricsService) Update(ctx context.Context, kind string, name string, value string) error {
	switch kind {
	case models.Gauge:
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return ErrBadRequest
		}
		return s.repo.SetGauge(ctx, name, v)
	case models.Counter:
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return ErrBadRequest
		}
		return s.repo.AddCounter(ctx, name, v)
	default:
		return ErrBadRequest
	}
}

func (s *MetricsService) JSONUpdate(ctx context.Context, metric models.Metrics) error {
	switch metric.MType {
	case models.Gauge:
		if metric.Value == nil {
			return ErrBadRequest
		}
		return s.repo.SetGauge(ctx, metric.ID, *metric.Value)
	case models.Counter:
		if metric.Delta == nil {
			return ErrBadRequest
		}
		return s.repo.AddCounter(ctx, metric.ID, *metric.Delta)
	default:
		return ErrBadRequest
	}
}

func (s *MetricsService) JSONGet(ctx context.Context, metric models.Metrics) (models.Metrics, error) {
	switch metric.MType {
	case models.Gauge:
		value, err := s.repo.GetGauge(ctx, metric.ID)
		if err != nil {
			return models.Metrics{}, err
		}
		return models.Metrics{ID: metric.ID, MType: metric.MType, Value: &value}, nil
	case models.Counter:
		delta, err := s.repo.GetCounter(ctx, metric.ID)
		if err != nil {
			return models.Metrics{}, err
		}
		return models.Metrics{ID: metric.ID, MType: metric.MType, Delta: &delta}, nil
	default:
		return models.Metrics{}, ErrBadRequest
	}
}

func (s *MetricsService) BatchUpdate(ctx context.Context, metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	for _, metric := range metrics {
		switch metric.MType {
		case models.Gauge:
			if metric.Value == nil {
				return ErrBadRequest
			}
		case models.Counter:
			if metric.Delta == nil {
				return ErrBadRequest
			}
		default:
			return ErrBadRequest
		}
	}

	return s.repo.UpdateBatch(ctx, metrics)
}
