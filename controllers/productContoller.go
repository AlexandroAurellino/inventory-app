package controllers

import (
	"database/sql"
	"inventory-app/config"
	"inventory-app/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CreateProduct handles creating a new product
// CreateProduct handles creating a new product
func CreateProduct(c *gin.Context) {
	var product models.Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	stmt, err := config.DB.Prepare(`
		INSERT INTO products (code, name, description, unit, category, created_at)
		VALUES (?, ?, ?, ?, ?, datetime('now'))
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer stmt.Close()

	result, err := stmt.Exec(product.Code, product.Name, product.Description, product.Unit, product.Category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	product.ID = int(id)

	// Initialize inventory summary row
	_, err = config.DB.Exec(`
		INSERT INTO inventory_summary (product_id, opening_stock, total_in, total_out, ending_stock, average_price, low_stock_threshold)
		VALUES (?, 0, 0, 0, 0, 0, 5)
	`, product.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize inventory summary"})
		return
	}

	c.JSON(http.StatusCreated, product)
}

// ListProducts retrieves all products
func ListProducts(c *gin.Context) {
	rows, err := config.DB.Query("SELECT id, code, name, description, unit, category FROM products")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.ID, &p.Code, &p.Name, &p.Description, &p.Unit, &p.Category); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		products = append(products, p)
	}

	c.JSON(http.StatusOK, products)
}

// GetProductByID retrieves a product by ID
func GetProductByID(c *gin.Context) {
	id := c.Param("id")

	var product models.Product
	err := config.DB.QueryRow("SELECT id, code, name, description, unit, category FROM products WHERE id = ?", id).
		Scan(&product.ID, &product.Code, &product.Name, &product.Description, &product.Unit, &product.Category)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, product)
}

// UpdateProduct handles updating an existing product
func UpdateProduct(c *gin.Context) {
	id := c.Param("id")
	var product models.Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := config.DB.Exec(`
		UPDATE products SET code = ?, name = ?, description = ?, unit = ?, category = ?
		WHERE id = ?`, product.Code, product.Name, product.Description, product.Unit, product.Category, id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product updated successfully"})
}

// DeleteProduct handles deleting a product
func DeleteProduct(c *gin.Context) {
	id := c.Param("id")

	_, err := config.DB.Exec("DELETE FROM products WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product deleted successfully"})
}
