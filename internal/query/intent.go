package query

import (
	"fmt"
	"regexp"
	"strings"
)

type QueryIntent struct {
	Intent       string            `json:"intent"`
	Entities     map[string]string `json:"entities"`
	RequiredData []string          `json:"required_data"`
}

var allowedIntents = map[string]bool{
	"get_order_status": true, "get_payment_status": true, "get_delivery_status": true,
	"get_order_summary": true, "diagnose_order_issue": true, "find_orders": true,
	"general_operations_query": true, "unknown": true,
}
var allowedData = map[string]bool{"order": true, "payment": true, "delivery": true, "customer": true}
var idPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
var orderIDPattern = regexp.MustCompile(`(?i)(?:order\s*(?:id|number|no)?\s*[:#]?\s*|#)([0-9]{1,64})`)

func InferIntent(text string) QueryIntent {
	lower := strings.ToLower(text)
	intent := "unknown"
	data := []string(nil)
	switch {
	case strings.Contains(lower, "payment") || strings.Contains(lower, "paid"):
		intent, data = "get_payment_status", []string{"payment"}
	case strings.Contains(lower, "delivery") || strings.Contains(lower, "delivered") || strings.Contains(lower, "shipping"):
		intent, data = "get_delivery_status", []string{"delivery"}
	case strings.Contains(lower, "summary") || strings.Contains(lower, "full status"):
		intent, data = "get_order_summary", []string{"order", "payment", "delivery", "customer"}
	case strings.Contains(lower, "why") || strings.Contains(lower, "issue") || strings.Contains(lower, "problem"):
		intent, data = "diagnose_order_issue", []string{"order", "payment", "delivery"}
	case strings.Contains(lower, "order") || strings.Contains(lower, "status"):
		intent, data = "get_order_status", []string{"order"}
	}
	entities := map[string]string{}
	if match := orderIDPattern.FindStringSubmatch(text); len(match) == 2 {
		entities["order_id"] = match[1]
	}
	return QueryIntent{Intent: intent, Entities: entities, RequiredData: data}
}

func NormalizeIntent(intent QueryIntent) QueryIntent {
	entities := make(map[string]string, len(intent.Entities))
	for key, value := range intent.Entities {
		switch key {
		case "order_number", "order_no", "orderNumber":
			key = "order_id"
		case "customer_number", "customer_no", "customerNumber":
			key = "customer_id"
		}
		entities[key] = value
	}
	intent.Entities = entities

	filtered := make([]string, 0, len(intent.RequiredData))
	for _, source := range intent.RequiredData {
		if source != "order_id" && source != "customer_id" {
			filtered = append(filtered, source)
		}
	}
	if required, ok := requiredDataByIntent[intent.Intent]; ok {
		filtered = required
	}
	intent.RequiredData = filtered
	return intent
}

var requiredDataByIntent = map[string][]string{
	"get_order_status":     {"order"},
	"get_payment_status":   {"payment"},
	"get_delivery_status":  {"delivery"},
	"get_order_summary":    {"order", "payment", "delivery", "customer"},
	"diagnose_order_issue": {"order", "payment", "delivery"},
	"find_orders":          {"order", "payment", "delivery"},
}

func ValidateIntent(intent QueryIntent) error {
	if !allowedIntents[intent.Intent] {
		return fmt.Errorf("unsupported intent %q", intent.Intent)
	}
	for source := range intent.RequiredData {
		if !allowedData[intent.RequiredData[source]] {
			return fmt.Errorf("unsupported data source %q", intent.RequiredData[source])
		}
	}
	for key, value := range intent.Entities {
		if key != "order_id" && key != "customer_id" {
			return fmt.Errorf("unsupported entity %q", key)
		}
		if !idPattern.MatchString(value) {
			return fmt.Errorf("invalid %s", key)
		}
	}
	return nil
}

func NeedsOrder(intent QueryIntent) bool {
	for _, source := range intent.RequiredData {
		if source == "order" || source == "payment" || source == "delivery" {
			return true
		}
	}
	return intent.Intent != "unknown" && intent.Intent != "general_operations_query"
}
