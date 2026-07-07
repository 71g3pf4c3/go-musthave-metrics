package repository

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsPostgresRetriableError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "connection exception class 08",
			err: &pgconn.PgError{
				Code: "08006",
			},
			want: true,
		},
		{
			name: "non connection error",
			err: &pgconn.PgError{
				Code: "23505",
			},
			want: false,
		},
		{
			name: "plain error",
			err:  errors.New("some error"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPostgresRetriableError(tt.err)
			if got != tt.want {
				t.Fatalf("isPostgresRetriableError()=%v, want=%v", got, tt.want)
			}
		})
	}
}
