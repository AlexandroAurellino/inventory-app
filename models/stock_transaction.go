// models/stock_transaction.go
package models

import "time"

type StockTransaction struct {
    ID                   int       `json:"id"`
    ProductID            int       `json:"product_id"`
    TransactionType      string    `json:"transaction_type"` // "in" or "out"
    Quantity             float64   `json:"quantity"`
    PricePerUnit         float64   `json:"price_per_unit"`
    TotalValue           float64   `json:"total_value"`
    Department           string    `json:"department"`
    TransactionTimestamp time.Time `json:"transaction_timestamp"`
    Notes                string    `json:"notes"`
}
