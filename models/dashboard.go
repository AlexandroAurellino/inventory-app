// models/dashboard.go
package models

type DashboardSummary struct {
    ProductCount      int          `json:"product_count"`
    LowStockCount     int          `json:"low_stock_count"`
    TransactionsToday int          `json:"transactions_today"`
    TopProducts       []TopProduct `json:"top_products"`
}

type TopProduct struct {
    Name             string `json:"name"`
    TransactionCount int    `json:"transaction_count"`
}