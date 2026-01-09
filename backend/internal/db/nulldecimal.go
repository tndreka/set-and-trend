package db

import (
	"database/sql/driver"
	"github.com/shopspring/decimal"
)

// NullDecimal represents a decimal.Decimal that may be null
type NullDecimal struct {
	Decimal decimal.Decimal
	Valid   bool // Valid is true if Decimal is not NULL
}

// Scan implements the Scanner interface
func (nd *NullDecimal) Scan(value interface{}) error {
	if value == nil {
		nd.Decimal, nd.Valid = decimal.Zero, false
		return nil
	}
	nd.Valid = true
	return nd.Decimal.Scan(value)
}

// Value implements the driver Valuer interface
func (nd NullDecimal) Value() (driver.Value, error) {
	if !nd.Valid {
		return nil, nil
	}
	return nd.Decimal.Value()
}
