package controllers

import (
	"database/sql"
	"inventory-app/config"
	"inventory-app/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const LOW_STOCK_THRESHOLD = 5 // configurable threshold for low-stock alerts

// CreateStockTransaction handles adding a new stock transaction and updating inventory summary.
func CreateStockTransaction(c *gin.Context) {
	var transaction models.StockTransaction

	if err := c.ShouldBindJSON(&transaction); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if transaction.Quantity <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Quantity must be greater than 0"})
		return
	}

	if transaction.TransactionType != "in" && transaction.TransactionType != "out" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction type"})
		return
	}

	if transaction.TransactionTimestamp.IsZero() {
		transaction.TransactionTimestamp = time.Now()
	}

	if transaction.TotalValue == 0 && transaction.PricePerUnit != 0 {
		transaction.TotalValue = transaction.PricePerUnit * transaction.Quantity
	}

	stmt, err := config.DB.Prepare(`
		INSERT INTO stock_transactions
		(product_id, transaction_type, quantity, price_per_unit, total_value, department, transaction_timestamp, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer stmt.Close()

	result, err := stmt.Exec(
		transaction.ProductID,
		transaction.TransactionType,
		transaction.Quantity,
		transaction.PricePerUnit,
		transaction.TotalValue,
		transaction.Department,
		transaction.TransactionTimestamp,
		transaction.Notes,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	id, _ := result.LastInsertId()
	transaction.ID = int(id)

	// --- Inventory Summary Handling ---
	var (
		currentIn, currentOut, currentAvg, currentEnding float64
		openingStock                                      float64
	)

	row := config.DB.QueryRow(`
		SELECT opening_stock, total_in, total_out, average_price, ending_stock 
		FROM inventory_summary WHERE product_id = ?
	`, transaction.ProductID)

	err = row.Scan(&openingStock, &currentIn, &currentOut, &currentAvg, &currentEnding)

	if err == sql.ErrNoRows {
		// First transaction for this product, initialize summary
		var stockIn, stockOut, avgPrice, ending float64
		if transaction.TransactionType == "in" {
			stockIn = transaction.Quantity
			avgPrice = transaction.PricePerUnit
			ending = stockIn
		} else {
			stockOut = transaction.Quantity
			ending = -stockOut
		}
		_, err = config.DB.Exec(`
			INSERT INTO inventory_summary
			(product_id, opening_stock, total_in, total_out, ending_stock, average_price)
			VALUES (?, ?, ?, ?, ?, ?)`,
			transaction.ProductID, 0.0, stockIn, stockOut, ending, avgPrice)
	} else if err == nil {
		// Update existing summary
		newIn := currentIn
		newOut := currentOut
		newEnding := currentEnding
		newAvg := currentAvg

		if transaction.TransactionType == "in" {
			newIn += transaction.Quantity
			newEnding += transaction.Quantity
			if transaction.PricePerUnit > 0 {
				newAvg = ((currentAvg * currentIn) + (transaction.PricePerUnit * transaction.Quantity)) / newIn
			}
		} else {
			newOut += transaction.Quantity
			newEnding -= transaction.Quantity
		}

		_, err = config.DB.Exec(`
			UPDATE inventory_summary
			SET total_in = ?, total_out = ?, ending_stock = ?, average_price = ?
			WHERE product_id = ?`,
			newIn, newOut, newEnding, newAvg, transaction.ProductID)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed updating inventory summary: " + err.Error()})
		return
	}

	// Optional: Low stock alert flag
	var isLowStock bool
	if currentEnding-transaction.Quantity <= LOW_STOCK_THRESHOLD && transaction.TransactionType == "out" {
		isLowStock = true
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":        "Transaction recorded successfully",
		"transaction":    transaction,
		"low_stock_alert": isLowStock,
	})
}

// ListStockTransactions retrieves all stock transactions.
func ListStockTransactions(c *gin.Context) {
	rows, err := config.DB.Query(`
		SELECT id, product_id, transaction_type, quantity, price_per_unit, total_value, department, transaction_timestamp, notes 
		FROM stock_transactions ORDER BY transaction_timestamp DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	transactions := []models.StockTransaction{}
	for rows.Next() {
		var t models.StockTransaction
		if err := rows.Scan(
			&t.ID, &t.ProductID, &t.TransactionType, &t.Quantity,
			&t.PricePerUnit, &t.TotalValue, &t.Department,
			&t.TransactionTimestamp, &t.Notes,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		transactions = append(transactions, t)
	}

	c.JSON(http.StatusOK, transactions)
}

func GetTransactionsByDate(c *gin.Context) {
	date := c.Query("date")
	if date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing date parameter"})
		return
	}

	// Validate date format
	_, err := time.Parse("2006-01-02", date)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format. Use YYYY-MM-DD"})
		return
	}

	rows, err := config.DB.Query(`
		SELECT 
			st.id, p.name, st.transaction_type, st.quantity, st.transaction_timestamp
		FROM 
			stock_transactions st
		JOIN 
			products p ON p.id = st.product_id
		WHERE 
			strftime('%Y-%m-%d', st.transaction_timestamp) = ?
		ORDER BY 
			st.transaction_timestamp DESC
	`, date)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type Transaction struct {
		ID         int       `json:"id"`
		Product    string    `json:"product"`
		Type       string    `json:"transaction_type"`
		Quantity   float64   `json:"quantity"`
		Timestamp  time.Time `json:"transaction_timestamp"`
	}

	var results []Transaction
	for rows.Next() {
		var tx Transaction
		if err := rows.Scan(&tx.ID, &tx.Product, &tx.Type, &tx.Quantity, &tx.Timestamp); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		results = append(results, tx)
	}

	c.JSON(http.StatusOK, results)
}
