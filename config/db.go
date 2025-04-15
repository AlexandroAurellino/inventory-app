package config

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

// InitDB initializes the SQLite database connection and creates necessary tables.
func InitDB() {
	var err error
	DB, err = sql.Open("sqlite", "./inventory.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	// Create tables if they do not exist.
	createTables()
}

func createTables() {
	// Begin a transaction for the table creation process
	tx, err := DB.Begin()
	if err != nil {
		log.Fatalf("Failed to begin transaction: %v", err)
	}

	// Creating products table
	productTable := `
		CREATE TABLE IF NOT EXISTS products (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		code TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		description TEXT,        -- Added description
		unit TEXT NOT NULL,
		category TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	_, err = tx.Exec(productTable)
	if err != nil {
		tx.Rollback() // Rollback in case of error
		log.Fatalf("Failed to create products table: %v", err)
	}

	// Creating stock_transactions table
	transactionTable := `
		CREATE TABLE IF NOT EXISTS stock_transactions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		product_id INTEGER,
		transaction_type TEXT CHECK(transaction_type IN ('in','out')) NOT NULL,
		quantity REAL NOT NULL,
		price_per_unit REAL,                -- New field for price per unit
		total_value REAL,                   -- New field: could be computed (quantity * price_per_unit)
		department TEXT,                    -- You may still wish to record this if needed
		transaction_timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		notes TEXT,
		FOREIGN KEY(product_id) REFERENCES products(id) ON DELETE CASCADE
	);`
	_, err = tx.Exec(transactionTable)
	if err != nil {
		tx.Rollback() // Rollback in case of error
		log.Fatalf("Failed to create transactions table: %v", err)
	}

	// Creating inventory_summary table
	inventorySummaryTable := `
		CREATE TABLE IF NOT EXISTS inventory_summary (
			product_id INTEGER PRIMARY KEY,
			opening_stock REAL DEFAULT 0,
			total_in REAL DEFAULT 0,
			total_out REAL DEFAULT 0,
			ending_stock REAL DEFAULT 0,
			average_price REAL DEFAULT 0,
			low_stock_threshold REAL DEFAULT 5,
			FOREIGN KEY(product_id) REFERENCES products(id) ON DELETE CASCADE
		);`
	_, err = tx.Exec(inventorySummaryTable)
	if err != nil {
		tx.Rollback() // Rollback in case of error
		log.Fatalf("Failed to create inventory_summary table: %v", err)
	}

	// Commit the transaction once all tables are created successfully
	err = tx.Commit()
	if err != nil {
		log.Fatalf("Failed to commit transaction: %v", err)
	}

	log.Println("Database tables created successfully.")
}
