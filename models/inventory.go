package models

type MonthlyInventoryReport struct {
	ProductID    int     `json:"product_id"`
	Code         string  `json:"code"`
	Name         string  `json:"name"`
	Unit         string  `json:"unit"`
	Category     string  `json:"category"`
	OpeningStock float64 `json:"opening_stock"`
	StockIn      float64 `json:"stock_in"`
	StockOut     float64 `json:"stock_out"`
	EndingStock  float64 `json:"ending_stock"`
	AveragePrice float64 `json:"average_price"`
	TotalValue   float64 `json:"total_value"`
}
