package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/dede999/tools_crud_go/internal/domain"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Create(ctx context.Context, d *domain.Device) error {
	query := `
		INSERT INTO devices (id, name, brand, state, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.ExecContext(ctx, query, d.ID, d.Name, d.Brand, string(d.State), d.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert device: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id string) (*domain.Device, error) {
	query := `
		SELECT id, name, brand, state, created_at
		FROM devices
		WHERE id = $1
	`
	var d domain.Device
	var stateStr string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&d.ID,
		&d.Name,
		&d.Brand,
		&stateStr,
		&d.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrDeviceNotFound
		}
		return nil, fmt.Errorf("failed to query device by id: %w", err)
	}

	d.State = domain.DeviceState(stateStr)
	return &d, nil
}

func (r *PostgresRepository) List(ctx context.Context, brand *string, state *domain.DeviceState) ([]domain.Device, error) {
	query := `SELECT id, name, brand, state, created_at FROM devices WHERE 1=1`
	var args []any
	argIdx := 1

	if brand != nil && *brand != "" {
		query += fmt.Sprintf(" AND brand = $%d", argIdx)
		args = append(args, *brand)
		argIdx++
	}

	if state != nil && *state != "" {
		query += fmt.Sprintf(" AND state = $%d", argIdx)
		args = append(args, string(*state))
		argIdx++
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list devices: %w", err)
	}
	defer rows.Close()

	devices := make([]domain.Device, 0)
	for rows.Next() {
		var d domain.Device
		var stateStr string

		if err := rows.Scan(&d.ID, &d.Name, &d.Brand, &stateStr, &d.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan device row: %w", err)
		}

		d.State = domain.DeviceState(stateStr)
		devices = append(devices, d)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating device rows: %w", err)
	}

	return devices, nil
}

func (r *PostgresRepository) Update(ctx context.Context, d *domain.Device) error {
	query := `
		UPDATE devices
		SET name = $1, brand = $2, state = $3
		WHERE id = $4
	`
	res, err := r.db.ExecContext(ctx, query, d.Name, d.Brand, string(d.State), d.ID)
	if err != nil {
		return fmt.Errorf("failed to update device: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrDeviceNotFound
	}

	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM devices WHERE id = $1`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete device: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrDeviceNotFound
	}

	return nil
}
