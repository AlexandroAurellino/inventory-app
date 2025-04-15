package controllers

import (
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

    // Basic validation
    if transaction.Quantity <= 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Quantity must be greater than 0"})
        return
    }

    if transaction.TransactionType != "in" && transaction.TransactionType != "out" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction type"})
        return
    }

    // For stock-out, verify there's enough stock
    if transaction.TransactionType == "out" {
        var currentStock float64
        err := config.DB.QueryRow(`
            SELECT ending_stock FROM inventory_summary WHERE product_id = ?
        `, transaction.ProductID).Scan(&currentStock)
        
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check current stock: " + err.Error()})
            return
        }
        
        if currentStock < transaction.Quantity {
            c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient stock for this transaction"})
            return
        }
    }

    // Set default timestamp if not provided
    if transaction.TransactionTimestamp.IsZero() {
        transaction.TransactionTimestamp = time.Now()
    }

    // Auto-calculate total value if needed
    if transaction.TotalValue == 0 && transaction.PricePerUnit > 0 {
        transaction.TotalValue = transaction.PricePerUnit * transaction.Quantity
    }

    // Start a transaction for atomicity
    tx, err := config.DB.Begin()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to begin transaction: " + err.Error()})
        return
    }

    // Insert the stock transaction
    stmt, err := tx.Prepare(`
        INSERT INTO stock_transactions
        (product_id, transaction_type, quantity, price_per_unit, total_value, department, transaction_timestamp, notes)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)
    `)
    if err != nil {
        tx.Rollback()
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
        tx.Rollback()
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    id, _ := result.LastInsertId()
    transaction.ID = int(id)

    // Get current inventory summary state
    var (
        currentIn, currentOut, currentAvg, currentEnding float64
    )

    row := tx.QueryRow(`
        SELECT total_in, total_out, average_price, ending_stock 
        FROM inventory_summary WHERE product_id = ?
    `, transaction.ProductID)

    err = row.Scan(&currentIn, &currentOut, &currentAvg, &currentEnding)
    if err != nil {
        tx.Rollback()
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get current inventory state: " + err.Error()})
        return
    }

    // Update inventory summary based on transaction type
    var newIn, newOut, newEnding, newAvg float64
    newIn = currentIn
    newOut = currentOut
    newEnding = currentEnding
    newAvg = currentAvg

    if transaction.TransactionType == "in" {
        newIn += transaction.Quantity
        newEnding += transaction.Quantity
        
        // Recalculate average price (weighted average) for stock-in
        if transaction.PricePerUnit > 0 {
            // Weighted average calculation
            totalValueBefore := currentAvg * currentIn
            totalValueNew := transaction.PricePerUnit * transaction.Quantity
            if newIn > 0 { // Avoid division by zero
                newAvg = (totalValueBefore + totalValueNew) / newIn
            }
        }
    } else { // "out"
        newOut += transaction.Quantity
        newEnding -= transaction.Quantity
        // Average price stays the same for stock-out
    }

    // Update the inventory summary
    _, err = tx.Exec(`
        UPDATE inventory_summary
        SET total_in = ?, total_out = ?, ending_stock = ?, average_price = ?
        WHERE product_id = ?
    `, newIn, newOut, newEnding, newAvg, transaction.ProductID)

    if err != nil {
        tx.Rollback()
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update inventory summary: " + err.Error()})
        return
    }

    // Check if this puts the product in low stock state
    var lowStockThreshold float64
    err = tx.QueryRow(`
        SELECT low_stock_threshold FROM inventory_summary WHERE product_id = ?
    `, transaction.ProductID).Scan(&lowStockThreshold)
    
    if err != nil {
        tx.Rollback()
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check low stock threshold: " + err.Error()})
        return
    }
    
    isLowStock := newEnding <= lowStockThreshold

    // Commit the transaction
    if err := tx.Commit(); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction: " + err.Error()})
        return
    }

    c.JSON(http.StatusCreated, gin.H{
        "message": "Transaction recorded successfully",
        "transaction": transaction,
        "inventory_update": gin.H{
            "previous_stock": currentEnding,
            "current_stock": newEnding,
            "average_price": newAvg,
        },
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
