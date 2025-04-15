package models

type InventorySummaryResponse struct {
	ProductID    int     `json:"product_id"`
	Code         string  `json:"code"`
	Name         string  `json:"name"`
	Unit         string  `json:"unit"`
	Category     string  `json:"category"`
	OpeningStock float64 `json:"opening_stock"`
	TotalIn      float64 `json:"total_in"`
	TotalOut     float64 `json:"total_out"`
	EndingStock  float64 `json:"ending_stock"`
	AveragePrice float64 `json:"average_price"`
}
