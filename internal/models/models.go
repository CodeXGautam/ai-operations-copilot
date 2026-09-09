package models

import "time"

type Customer struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Phone     *string   `json:"phone,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Order struct {
	ID              string    `json:"id"`
	CustomerID      string    `json:"customer_id"`
	Status          string    `json:"status"`
	TotalAmount     int64     `json:"total_amount"`
	Currency        string    `json:"currency"`
	RejectionReason *string   `json:"rejection_reason,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type Payment struct {
	ID            string     `json:"id"`
	OrderID       string     `json:"order_id"`
	Status        string     `json:"status"`
	Amount        int64      `json:"amount"`
	TransactionID *string    `json:"transaction_id,omitempty"`
	PaidAt        *time.Time `json:"paid_at,omitempty"`
}

type Delivery struct {
	ID          string     `json:"id"`
	OrderID     string     `json:"order_id"`
	Status      string     `json:"status"`
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
	Courier     *string    `json:"courier,omitempty"`
	TrackingID  *string    `json:"tracking_id,omitempty"`
}
