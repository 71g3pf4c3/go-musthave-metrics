package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/mocks"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/models"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/service"
	"github.com/golang/mock/gomock"
)

func TestUpdateGaugeCallsRepository(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepository(ctrl)
	svc := service.New(repo)

	repo.EXPECT().SetGauge(gomock.Any(), "cpu", 42.5).Return(nil)

	err := svc.Update(context.Background(), models.Gauge, "cpu", "42.5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateCounterBadRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepository(ctrl)
	svc := service.New(repo)

	err := svc.Update(context.Background(), models.Counter, "hits", "not-int")
	if !errors.Is(err, service.ErrBadRequest) {
		t.Fatalf("expected ErrBadRequest, got: %v", err)
	}
}

func TestJSONGetGaugeFromRepository(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepository(ctrl)
	svc := service.New(repo)

	repo.EXPECT().GetGauge(gomock.Any(), "cpu").Return(12.34, nil)

	resp, err := svc.JSONGet(context.Background(), models.Metrics{ID: "cpu", MType: models.Gauge})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Value == nil || *resp.Value != 12.34 {
		t.Fatalf("unexpected value: %+v", resp)
	}
}

func TestListRepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepository(ctrl)
	svc := service.New(repo)

	repo.EXPECT().GetAllGauge(gomock.Any()).Return(nil, errors.New("db error"))

	_, _, err := svc.List(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
