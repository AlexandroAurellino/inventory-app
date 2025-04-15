package routes

import (
	"inventory-app/controllers"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api")
	{
		// Product routes
		api.POST("/products", controllers.CreateProduct)
		api.GET("/products", controllers.ListProducts)
		api.GET("/products/:id", controllers.GetProductByID)   // For GET Product by ID
		api.PUT("/products/:id", controllers.UpdateProduct)    // For PUT Update Product
		api.DELETE("/products/:id", controllers.DeleteProduct) // For DELETE Product

		// Transaction routes
		api.POST("/transactions", controllers.CreateStockTransaction)
		api.GET("/transactions", controllers.ListStockTransactions)
		api.GET("/transactions/by-date", controllers.GetTransactionsByDate)

		// Inventory routes
		api.GET("/inventory/summary", controllers.GetInventorySummary)
		api.GET("/inventory/summary/monthly", controllers.GetMonthlyInventorySummary) // Keep only one registration
		api.GET("/inventory/low-stock", controllers.GetLowStockAlerts)
		api.PUT("/inventory/:id/threshold", controllers.UpdateLowStockThreshold)
	}
}
