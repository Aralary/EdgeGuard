package postgres

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsUniqueViolation(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "unique violation",
			err:  &pgconn.PgError{Code: uniqueViolationCode},
			want: true,
		},
		{
			name: "different postgres error",
			err:  &pgconn.PgError{Code: "23503"},
			want: false,
		},
		{
			name: "wrapped unique violation",
			err:  errors.Join(errors.New("insert failed"), &pgconn.PgError{Code: uniqueViolationCode}),
			want: true,
		},
		{
			name: "ordinary error",
			err:  errors.New("boom"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isUniqueViolation(tt.err); got != tt.want {
				t.Fatalf("isUniqueViolation() = %v, want %v", got, tt.want)
			}
		})
	}
}
