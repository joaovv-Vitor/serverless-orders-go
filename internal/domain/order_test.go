package domain

import "testing"

func TestCreateOrderInputValidate(t *testing.T) {
	t.Parallel()

	validInput := CreateOrderInput{
		CustomerID: "customer-123",
		Items: []OrderItem{
			{ProductID: "product-456", Quantity: 2},
		},
	}

	tests := []struct {
		name    string
		input   CreateOrderInput
		wantErr string
	}{
		{name: "valid order", input: validInput},
		{
			name:    "missing customer",
			input:   CreateOrderInput{Items: validInput.Items},
			wantErr: "customerId is required",
		},
		{
			name:    "blank customer",
			input:   CreateOrderInput{CustomerID: "  ", Items: validInput.Items},
			wantErr: "customerId is required",
		},
		{
			name:    "missing items",
			input:   CreateOrderInput{CustomerID: validInput.CustomerID},
			wantErr: "at least one item is required",
		},
		{
			name: "missing product",
			input: CreateOrderInput{
				CustomerID: validInput.CustomerID,
				Items:      []OrderItem{{Quantity: 1}},
			},
			wantErr: "items[0].productId is required",
		},
		{
			name: "zero quantity",
			input: CreateOrderInput{
				CustomerID: validInput.CustomerID,
				Items:      []OrderItem{{ProductID: "product-456"}},
			},
			wantErr: "items[0].quantity must be greater than zero",
		},
		{
			name: "negative quantity",
			input: CreateOrderInput{
				CustomerID: validInput.CustomerID,
				Items:      []OrderItem{{ProductID: "product-456", Quantity: -1}},
			},
			wantErr: "items[0].quantity must be greater than zero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.input.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v", err)
				}
				return
			}
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("Validate() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}
