package query

import (
	"context"
	"fmt"

	"ai-operations-copilot/internal/models"
	"ai-operations-copilot/internal/repo"
)

type OperationalContext struct {
	Order        any    `json:"order,omitempty"`
	Payment      any    `json:"payment,omitempty"`
	Delivery     any    `json:"delivery,omitempty"`
	Customer     any    `json:"customer,omitempty"`
	Orders       any    `json:"orders,omitempty"`
	TotalMatches int    `json:"total_matches,omitempty"`
	Notice       string `json:"notice,omitempty"`
}

type ContextBuilder struct {
	Orders     repo.OrderRepository
	Payments   repo.PaymentRepository
	Deliveries repo.DeliveryRepository
	Customers  repo.CustomerRepository
}

func (b *ContextBuilder) Build(ctx context.Context, intent QueryIntent) (OperationalContext, error) {
	orderID := intent.Entities["order_id"]
	if intent.Intent == "find_orders" || (orderID == "" && !NeedsOrder(intent)) {
		filter := repo.OrderFilter{Limit: 20}
		if intent.Intent == "find_orders" {
			filter.PaymentStatus = stringPtr("paid")
			filter.DeliveryStatus = stringPtr("not_scheduled")
		}
		orders, err := b.Orders.Find(ctx, filter)
		if err != nil {
			return OperationalContext{}, err
		}
		return OperationalContext{Orders: orders, TotalMatches: len(orders)}, nil
	}
	if orderID == "" && NeedsOrder(intent) {
		return OperationalContext{Notice: "An order ID is required to retrieve operational records."}, nil
	}
	result := OperationalContext{}
	for _, source := range intent.RequiredData {
		switch source {
		case "order":
			value, err := b.Orders.GetByID(ctx, orderID)
			if err != nil {
				return result, fmt.Errorf("order lookup: %w", err)
			}
			result.Order = value
		case "payment":
			value, err := b.Payments.GetByOrderID(ctx, orderID)
			if err != nil {
				return result, fmt.Errorf("payment lookup: %w", err)
			}
			result.Payment = value
		case "delivery":
			value, err := b.Deliveries.GetByOrderID(ctx, orderID)
			if err != nil {
				return result, fmt.Errorf("delivery lookup: %w", err)
			}
			result.Delivery = value
		case "customer":
			if result.Order == nil {
				value, err := b.Orders.GetByID(ctx, orderID)
				if err != nil {
					return result, fmt.Errorf("order lookup: %w", err)
				}
				result.Order = value
			}
			order := result.Order.(*models.Order)
			value, err := b.Customers.GetByID(ctx, order.CustomerID)
			if err != nil {
				return result, fmt.Errorf("customer lookup: %w", err)
			}
			result.Customer = value
		}
	}
	return result, nil
}

func stringPtr(value string) *string {
	return &value
}
