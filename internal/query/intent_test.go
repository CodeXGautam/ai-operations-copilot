package query

import "testing"

func TestValidateIntent(t *testing.T) {
	valid := QueryIntent{Intent: "get_payment_status", Entities: map[string]string{"order_id": "4521"}, RequiredData: []string{"payment"}}
	if err := ValidateIntent(valid); err != nil {
		t.Fatalf("valid intent rejected: %v", err)
	}
	invalid := QueryIntent{Intent: "drop_database", Entities: map[string]string{}, RequiredData: []string{"order"}}
	if err := ValidateIntent(invalid); err == nil {
		t.Fatal("unsupported intent accepted")
	}
	invalidData := QueryIntent{Intent: "get_order_status", Entities: map[string]string{}, RequiredData: []string{"sql"}}
	if err := ValidateIntent(invalidData); err == nil {
		t.Fatal("unsupported data source accepted")
	}
	invalidID := QueryIntent{Intent: "get_order_status", Entities: map[string]string{"order_id": "1; DROP TABLE orders"}, RequiredData: []string{"order"}}
	if err := ValidateIntent(invalidID); err == nil {
		t.Fatal("invalid entity accepted")
	}
}

func TestNormalizeIntentDerivesDataSources(t *testing.T) {
	intent := NormalizeIntent(QueryIntent{
		Intent:       "get_payment_status",
		Entities:     map[string]string{"order_id": "4521"},
		RequiredData: []string{"order_id"},
	})
	if err := ValidateIntent(intent); err != nil {
		t.Fatal(err)
	}
	if len(intent.RequiredData) != 1 || intent.RequiredData[0] != "payment" {
		t.Fatalf("unexpected required data: %#v", intent.RequiredData)
	}
}

func TestNormalizeIntentMapsEntityAliases(t *testing.T) {
	intent := NormalizeIntent(QueryIntent{
		Intent:       "get_payment_status",
		Entities:     map[string]string{"order_number": "4521"},
		RequiredData: []string{"payment"},
	})
	if err := ValidateIntent(intent); err != nil {
		t.Fatal(err)
	}
	if intent.Entities["order_id"] != "4521" {
		t.Fatalf("order alias was not normalized: %#v", intent.Entities)
	}
}

func TestInferIntentFromSimplePaymentQuestion(t *testing.T) {
	intent := InferIntent("What's the payment status for order #4521?")
	if intent.Intent != "get_payment_status" || intent.Entities["order_id"] != "4521" {
		t.Fatalf("unexpected inferred intent: %#v", intent)
	}
}
