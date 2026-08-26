package domain

import (
	"errors"
	"fmt"
	"strings"
)

// OrderItem identifies a product and the amount requested by the customer.
type OrderItem struct {
	ProductID string `json:"productId"`
	Quantity  int    `json:"quantity"`
}

// CreateOrderInput contains the business data needed to accept an order.
type CreateOrderInput struct {
	CustomerID string      `json:"customerId"`
	Items      []OrderItem `json:"items"`
}

// Validate checks the minimum business rules introduced in phase 2.
func (input CreateOrderInput) Validate() error {
	if strings.TrimSpace(input.CustomerID) == "" {
		return errors.New("customerId is required")
	}
	if len(input.Items) == 0 {
		return errors.New("at least one item is required")
	}

	for index, item := range input.Items {
		if strings.TrimSpace(item.ProductID) == "" {
			return fmt.Errorf("items[%d].productId is required", index)
		}
		if item.Quantity <= 0 {
			return fmt.Errorf("items[%d].quantity must be greater than zero", index)
		}
	}

	return nil
}
