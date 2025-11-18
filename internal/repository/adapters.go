package repository

import (
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ttl256/metrics/database"
	models "github.com/ttl256/metrics/internal/model"
)

func toDomain(m database.Metric) models.Metrics {
	var delta *int64
	if m.Delta.Valid {
		delta = &m.Delta.Int64
	}
	var value *float64
	if m.Value.Valid {
		value = &m.Value.Float64
	}
	return models.Metrics{
		ID:    m.ID,
		MType: m.Type,
		Delta: delta,
		Value: value,
		Hash:  m.Hash.String,
	}
}

func toRepo(m models.Metrics) database.Metric {
	var delta pgtype.Int8
	if m.Delta == nil {
		delta = pgtype.Int8{} //nolint: exhaustruct //fine
	} else {
		delta = pgtype.Int8{
			Int64: *m.Delta,
			Valid: true,
		}
	}
	var value pgtype.Float8
	if m.Value == nil {
		value = pgtype.Float8{} //nolint: exhaustruct //fine
	} else {
		value = pgtype.Float8{
			Float64: *m.Value,
			Valid:   true,
		}
	}
	var hash pgtype.Text
	if m.Hash == "" {
		hash = pgtype.Text{} //nolint: exhaustruct //fine
	} else {
		hash = pgtype.Text{
			String: m.Hash,
			Valid:  true,
		}
	}
	return database.Metric{
		ID:    m.ID,
		Type:  m.MType,
		Delta: delta,
		Value: value,
		Hash:  hash,
	}
}
