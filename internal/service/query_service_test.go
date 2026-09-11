package service

import (
	"context"
	"testing"

	"ai-operations-copilot/internal/llm"
	"ai-operations-copilot/internal/models"
	"ai-operations-copilot/internal/query"
	"ai-operations-copilot/internal/repo"
)

type fakeLLM struct{ calls []llm.CompletionRequest }

func (f *fakeLLM) Complete(_ context.Context, request llm.CompletionRequest) (string, error) {
	f.calls = append(f.calls, request)
	if len(f.calls) == 1 {
		return `{"intent":"get_payment_status","entities":{"order_id":"4521"},"required_data":["payment"]}`, nil
	}
	return "Payment is paid.", nil
}
func (f *fakeLLM) Stream(_ context.Context, _ llm.CompletionRequest, callback func(string) error) error {
	return callback("Payment is paid.")
}

type orderRepo struct{}

func (orderRepo) GetByID(context.Context, string) (*models.Order, error) {
	return nil, repo.ErrNotFound
}
func (orderRepo) Find(context.Context, repo.OrderFilter) ([]models.Order, error) { return nil, nil }

type paymentRepo struct{}

func (paymentRepo) GetByOrderID(context.Context, string) (*models.Payment, error) {
	return &models.Payment{OrderID: "4521", Status: "paid", Amount: 72000}, nil
}

type deliveryRepo struct{}

func (deliveryRepo) GetByOrderID(context.Context, string) (*models.Delivery, error) {
	return nil, repo.ErrNotFound
}

type customerRepo struct{}

func (customerRepo) GetByID(context.Context, string) (*models.Customer, error) {
	return nil, repo.ErrNotFound
}

func TestProcessUsesIntentAndOperationalContext(t *testing.T) {
	client := &fakeLLM{}
	svc := &QueryService{LLM: client, Context: &query.ContextBuilder{Orders: orderRepo{}, Payments: paymentRepo{}, Deliveries: deliveryRepo{}, Customers: customerRepo{}}, MaxQueryLength: 4000}
	response, err := svc.Process(context.Background(), "What's the payment status for order #4521?")
	if err != nil {
		t.Fatal(err)
	}
	if response.Answer != "Payment is paid." || response.Intent != "get_payment_status" || response.OrderID != "4521" {
		t.Fatalf("unexpected response: %+v", response)
	}
	if len(client.calls) != 2 || client.calls[1].Messages[1].Content == "" {
		t.Fatal("response LLM did not receive context")
	}
}
