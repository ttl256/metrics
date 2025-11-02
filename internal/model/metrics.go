package models

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

const (
	Counter = "counter"
	Gauge   = "gauge"
)

type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
}

func (m *Metrics) UnmarshalJSON(data []byte) error {
	type MetricsAlias Metrics
	var mm MetricsAlias
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&mm); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	if mm.ID == "" {
		return errors.New("metrics ID is required")
	}
	if mm.MType != Counter && mm.MType != Gauge {
		return fmt.Errorf("unexpected metrics value %q", mm.MType)
	}
	*m = Metrics(mm)
	return nil
}
