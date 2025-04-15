package controllers

import (
	"inventory-app/config"
	"inventory-app/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// GetInventorySummary returns current inventory summary
func GetInventorySummary(c *gin.Context) {
	rows, err := config.DB.Query(`
		SELECT 
			p.id, p.code, p.name, i.opening_stock, i.total_in, i.total_out, i.ending_stock, i.average_price
		FROM 
			inventory_summary i
		JOIN 
			products p ON p.id = i.product_id
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type Inventory struct {
		ID            int     `json:"id"`
		Code          string  `json:"code"`
		Name          string  `json:"name"`
		OpeningStock  float64 `json:"opening_stock"`
		TotalIn       float64 `json:"total_in"`
		TotalOut      float64 `json:"total_out"`
		EndingStock   float64 `json:"ending_stock"`
		AveragePrice  float64 `json:"average_price"`
	}

	var summaries []Inventory
	for rows.Next() {
		var inv Inventory
		err := rows.Scan(&inv.ID, &inv.Code, &inv.Name, &inv.OpeningStock, &inv.TotalIn, &inv.TotalOut, &inv.EndingStock, &inv.AveragePrice)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		summaries = append(summaries, inv)
	}

	c.JSON(http.StatusOK, summaries)
}


// GetLowStockAlerts returns products with low stock
func GetLowStockAlerts(c *gin.Context) {
	rows, err := config.DB.Query(`
		SELECT 
			p.id, p.code, p.name, i.ending_stock, i.low_stock_threshold
		FROM 
			inventory_summary i
		JOIN 
			products p ON p.id = i.product_id
		WHERE 
			i.ending_stock < i.low_stock_threshold
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type Alert struct {
		ID              int     `json:"id"`
		Code            string  `json:"code"`
		Name            string  `json:"name"`
		EndingStock     float64 `json:"ending_stock"`
		LowStockThresh  float64 `json:"low_stock_threshold"`
	}

	var alerts []Alert
	for rows.Next() {
		var a Alert
		if err := rows.Scan(&a.ID, &a.Code, &a.Name, &a.EndingStock, &a.LowStockThresh); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		alerts = append(alerts, a)
	}

	c.JSON(http.StatusOK, alerts)
}

func GetMonthlyInventorySummary(c *gin.Context) {
	month := c.DefaultQuery("month", "") // default to empty string if no query param

	if month == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing month parameter (expected format: YYYY-MM)"})
		return
	}

	// Check if the month format is correct
	_, err := time.Parse("2006-01", month)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid month format. Expected format: YYYY-MM"})
		return
	}

	// Generate the start and end date for the month
	startDate := month + "-01"
	// This assumes the end date is the last day of the month
	endDate := month + "-31"

	// Query to get transactions for the specific month
	rows, err := config.DB.Query(`
		SELECT 
			p.id, p.code, p.name, p.unit, p.category,
			COALESCE(SUM(CASE WHEN st.transaction_type = 'in' THEN st.quantity ELSE 0 END), 0) AS stock_in,
			COALESCE(SUM(CASE WHEN st.transaction_type = 'out' THEN st.quantity ELSE 0 END), 0) AS stock_out
		FROM 
			products p
		LEFT JOIN 
			stock_transactions st ON p.id = st.product_id AND DATE(st.transaction_timestamp) BETWEEN ? AND ?
		GROUP BY 
			p.id
	`, startDate, endDate)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var results []models.InventorySummaryResponse
	for rows.Next() {
		var item models.InventorySummaryResponse
		err := rows.Scan(&item.ProductID, &item.Code, &item.Name, &item.Unit, &item.Category, &item.TotalIn, &item.TotalOut)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		results = append(results, item)
	}

	c.JSON(http.StatusOK, results)
}

func UpdateLowStockThreshold(c *gin.Context) {
	id := c.Param("id")

	type ThresholdUpdateRequest struct {
		NewThreshold float64 `json:"new_threshold"`
	}

	var request ThresholdUpdateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	_, err := config.DB.Exec(`
		UPDATE inventory_summary 
		SET low_stock_threshold = ? 
		WHERE product_id = ?`, request.NewThreshold, id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update threshold"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Threshold updated successfully"})
}
