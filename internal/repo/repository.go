package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"ai-operations-copilot/internal/models"
)

var ErrNotFound = errors.New("record not found")

type OrderFilter struct {
	PaymentStatus  *string
	DeliveryStatus *string
	OrderStatus    *string
	Limit          int
}

type OrderRepository interface {
	GetByID(context.Context, string) (*models.Order, error)
	Find(context.Context, OrderFilter) ([]models.Order, error)
}
type PaymentRepository interface {
	GetByOrderID(context.Context, string) (*models.Payment, error)
}
type DeliveryRepository interface {
	GetByOrderID(context.Context, string) (*models.Delivery, error)
}
type CustomerRepository interface {
	GetByID(context.Context, string) (*models.Customer, error)
}

type Store struct{ DB *sql.DB }

type OrderStore struct{ *Store }
type PaymentStore struct{ *Store }
type DeliveryStore struct{ *Store }
type CustomerStore struct{ *Store }

func NewStore(db *sql.DB) *Store { return &Store{DB: db} }

func (s *Store) Orders() *OrderStore        { return &OrderStore{s} }
func (s *Store) Payments() *PaymentStore    { return &PaymentStore{s} }
func (s *Store) Deliveries() *DeliveryStore { return &DeliveryStore{s} }
func (s *Store) Customers() *CustomerStore  { return &CustomerStore{s} }

func (s *OrderStore) GetByID(ctx context.Context, id string) (*models.Order, error) {
	return s.Store.GetByID(ctx, id)
}
func (s *OrderStore) Find(ctx context.Context, filter OrderFilter) ([]models.Order, error) {
	return s.Store.Find(ctx, filter)
}
func (s *PaymentStore) GetByOrderID(ctx context.Context, id string) (*models.Payment, error) {
	return s.Store.GetByOrderID(ctx, id)
}
func (s *DeliveryStore) GetByOrderID(ctx context.Context, id string) (*models.Delivery, error) {
	return s.Store.GetDeliveryByOrderID(ctx, id)
}
func (s *CustomerStore) GetByID(ctx context.Context, id string) (*models.Customer, error) {
	return s.Store.GetCustomerByID(ctx, id)
}

func (s *Store) GetByID(ctx context.Context, id string) (*models.Order, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT id, customer_id, status, total_amount, currency, rejection_reason, created_at FROM orders WHERE id = ?`, id)
	var order models.Order
	if err := row.Scan(&order.ID, &order.CustomerID, &order.Status, &order.TotalAmount, &order.Currency, &order.RejectionReason, &order.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get order: %w", err)
	}
	return &order, nil
}

func (s *Store) Find(ctx context.Context, filter OrderFilter) ([]models.Order, error) {
	conditions := []string{"1 = 1"}
	args := make([]any, 0, 3)
	if filter.OrderStatus != nil {
		conditions = append(conditions, "o.status = ?")
		args = append(args, *filter.OrderStatus)
	}
	if filter.PaymentStatus != nil {
		conditions = append(conditions, "p.status = ?")
		args = append(args, *filter.PaymentStatus)
	}
	if filter.DeliveryStatus != nil {
		conditions = append(conditions, "d.status = ?")
		args = append(args, *filter.DeliveryStatus)
	}
	limit := filter.Limit
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	query := `SELECT DISTINCT o.id, o.customer_id, o.status, o.total_amount, o.currency, o.rejection_reason, o.created_at FROM orders o LEFT JOIN payments p ON p.order_id = o.id LEFT JOIN deliveries d ON d.order_id = o.id WHERE ` + strings.Join(conditions, " AND ") + ` ORDER BY o.created_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("find orders: %w", err)
	}
	defer rows.Close()
	orders := make([]models.Order, 0)
	for rows.Next() {
		var order models.Order
		if err := rows.Scan(&order.ID, &order.CustomerID, &order.Status, &order.TotalAmount, &order.Currency, &order.RejectionReason, &order.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

func (s *Store) GetByOrderID(ctx context.Context, orderID string) (*models.Payment, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT id, order_id, status, amount, transaction_id, paid_at FROM payments WHERE order_id = ?`, orderID)
	var payment models.Payment
	if err := row.Scan(&payment.ID, &payment.OrderID, &payment.Status, &payment.Amount, &payment.TransactionID, &payment.PaidAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get payment: %w", err)
	}
	return &payment, nil
}

func (s *Store) GetDeliveryByOrderID(ctx context.Context, orderID string) (*models.Delivery, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT id, order_id, status, scheduled_at, delivered_at, courier, tracking_id FROM deliveries WHERE order_id = ?`, orderID)
	var delivery models.Delivery
	if err := row.Scan(&delivery.ID, &delivery.OrderID, &delivery.Status, &delivery.ScheduledAt, &delivery.DeliveredAt, &delivery.Courier, &delivery.TrackingID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get delivery: %w", err)
	}
	return &delivery, nil
}

func (s *Store) GetCustomerByID(ctx context.Context, id string) (*models.Customer, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT id, name, email, phone, created_at FROM customers WHERE id = ?`, id)
	var customer models.Customer
	if err := row.Scan(&customer.ID, &customer.Name, &customer.Email, &customer.Phone, &customer.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get customer: %w", err)
	}
	return &customer, nil
}
