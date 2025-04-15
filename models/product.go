// Updated models/product.go
package models

import "time"

type Product struct {
    ID          int       `json:"id"`
    Code        string    `json:"code"`
    Name        string    `json:"name"`
    Description string    `json:"description"` // Change from pointer to string
    Unit        string    `json:"unit"`
    Category    string    `json:"category"`
    CreatedAt   time.Time `json:"created_at"`
}