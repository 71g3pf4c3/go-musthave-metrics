package repository

import (
	"context"
	"database/sql"
	stderrs "errors"
	"fmt"
	"strconv"
	"time"

	"github.com/71g3pf4c3/go-musthave-metrics/internal/logger"
	"github.com/71g3pf4c3/go-musthave-metrics/internal/models"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	defaultMigrateAttempts = 20
	defaultMigrateTimeout  = time.Second
)

type PGStorage struct {
	db *sql.DB
}

func NewPGStorage(dsn string) (*PGStorage, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.PingContext(context.Background()); err != nil {
		return nil, err
	}

	if err := runMigrations(dsn); err != nil {
		return nil, err
	}

	logger.Sugar.Infof("connected to postgres")
	logger.Sugar.Debugf("connecting to postgres with dsn %s", dsn)

	return &PGStorage{db: db}, nil
}

func runMigrations(dsn string) error {
	databaseURL := dsn

	var (
		attempts = defaultMigrateAttempts
		err      error
		m        *migrate.Migrate
	)

	for attempts > 0 {
		m, err = migrate.New("file://migrations", databaseURL)
		if err == nil {
			break
		}

		logger.Sugar.Infof("migrate: postgres is trying to connect, attempts left: %d", attempts)
		time.Sleep(defaultMigrateTimeout)
		attempts--
	}

	if err != nil {
		return fmt.Errorf("migrate: postgres connect error: %w", err)
	}

	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil {
			logger.Sugar.Debugf("migrate: source close error: %v", srcErr)
		}
		if dbErr != nil {
			logger.Sugar.Debugf("migrate: database close error: %v", dbErr)
		}
	}()

	err = m.Up()
	if err != nil && !stderrs.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate: up error: %w", err)
	}

	if stderrs.Is(err, migrate.ErrNoChange) {
		logger.Sugar.Infof("migrate: no change")
		return nil
	}

	logger.Sugar.Infof("migrate: up success")
	return nil
}

func (p *PGStorage) Ping(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	return p.db.PingContext(ctx)
}

func (p *PGStorage) Close() error {
	return p.db.Close()
}

func (p *PGStorage) AddCounter(ctx context.Context, key string, value int64) error {
	_, err := p.db.ExecContext(ctx, `
		INSERT INTO metrics (name, kind, value_bigint)
		VALUES ($1, $2, $3)
		ON CONFLICT (name, kind)
		DO UPDATE SET value_bigint = metrics.value_bigint + EXCLUDED.value_bigint
	`, key, models.Counter, value)
	return err
}

func (p *PGStorage) SetGauge(ctx context.Context, key string, value float64) error {
	_, err := p.db.ExecContext(ctx, `
		INSERT INTO metrics (name, kind, value_double)
		VALUES ($1, $2, $3)
		ON CONFLICT (name, kind)
		DO UPDATE SET value_double = EXCLUDED.value_double
	`, key, models.Gauge, value)
	return err
}

func (p *PGStorage) GetValue(ctx context.Context, key string, kind string) (string, error) {
	switch kind {
	case models.Gauge:
		value, err := p.GetGauge(ctx, key)
		if err != nil {
			return "", err
		}
		return strconv.FormatFloat(value, 'f', -1, 64), nil
	case models.Counter:
		value, err := p.GetCounter(ctx, key)
		if err != nil {
			return "", err
		}
		return strconv.FormatInt(value, 10), nil
	default:
		return "", fmt.Errorf("UnexpectedError")
	}
}

func (p *PGStorage) GetGauge(ctx context.Context, key string) (float64, error) {
	var value float64
	err := p.db.QueryRowContext(ctx, `
		SELECT value_double FROM metrics WHERE name = $1 AND kind = $2
	`, key, models.Gauge).Scan(&value)
	if err != nil {
		if stderrs.Is(err, sql.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	return value, nil
}

func (p *PGStorage) GetCounter(ctx context.Context, key string) (int64, error) {
	var value int64
	err := p.db.QueryRowContext(ctx, `
		SELECT value_bigint FROM metrics WHERE name = $1 AND kind = $2
	`, key, models.Counter).Scan(&value)
	if err != nil {
		if stderrs.Is(err, sql.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	return value, nil
}

func (p *PGStorage) GetAllGauge(ctx context.Context) (map[string]float64, error) {
	rows, err := p.db.QueryContext(ctx, `
		SELECT name, value_double FROM metrics WHERE kind = $1
	`, models.Gauge)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]float64)
	for rows.Next() {
		var name string
		var value float64
		if err := rows.Scan(&name, &value); err != nil {
			return nil, err
		}
		result[name] = value
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (p *PGStorage) GetAllCounter(ctx context.Context) (map[string]int64, error) {
	rows, err := p.db.QueryContext(ctx, `
		SELECT name, value_bigint FROM metrics WHERE kind = $1
	`, models.Counter)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var name string
		var value int64
		if err := rows.Scan(&name, &value); err != nil {
			return nil, err
		}
		result[name] = value
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (p *PGStorage) Dump(ctx context.Context, path string) error {
	return nil
}

func (p *PGStorage) Restore(ctx context.Context, path string) error {
	return nil
}
