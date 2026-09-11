package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"ai-operations-copilot/internal/llm"
	"ai-operations-copilot/internal/query"
	"ai-operations-copilot/internal/repo"
)

type QueryResponse struct {
	Answer  string `json:"answer"`
	Intent  string `json:"intent,omitempty"`
	OrderID string `json:"order_id,omitempty"`
}
type QueryService struct {
	LLM            llm.Client
	Context        *query.ContextBuilder
	MaxQueryLength int
	LLMTimeout     time.Duration
}

func (s *QueryService) Prepare(ctx context.Context, text string) (query.QueryIntent, query.OperationalContext, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return query.QueryIntent{}, query.OperationalContext{}, ErrInvalidQuery
	}
	if len([]rune(text)) > s.MaxQueryLength {
		return query.QueryIntent{}, query.OperationalContext{}, ErrQueryTooLong
	}
	intentCtx, cancel := s.withLLMTimeout(ctx)
	defer cancel()
	intentJSON, err := s.LLM.Complete(intentCtx, llm.CompletionRequest{Messages: []llm.Message{{Role: "system", Content: llm.IntentSystemPrompt}, {Role: "user", Content: text}}, JSONMode: true})
	if err != nil {
		return query.QueryIntent{}, query.OperationalContext{}, fmt.Errorf("intent completion: %w", err)
	}
	var intent query.QueryIntent
	decoder := json.NewDecoder(strings.NewReader(intentJSON))
	decoder.DisallowUnknownFields()
	decodeErr := decoder.Decode(&intent)
	if intent.Entities == nil {
		intent.Entities = map[string]string{}
	}
	inferred := query.InferIntent(text)
	if intent.Entities["order_id"] == "" && inferred.Entities["order_id"] != "" {
		intent.Entities["order_id"] = inferred.Entities["order_id"]
	}
	if decodeErr != nil {
		intent = inferred
	}
	intent = query.NormalizeIntent(intent)
	if err := query.ValidateIntent(intent); err != nil {
		return intent, query.OperationalContext{}, fmt.Errorf("invalid intent: %w", err)
	}
	data, err := s.Context.Build(ctx, intent)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return intent, query.OperationalContext{}, ErrRecordNotFound
		}
		return intent, query.OperationalContext{}, err
	}
	return intent, data, nil
}

func (s *QueryService) Process(ctx context.Context, text string) (QueryResponse, error) {
	intent, data, err := s.Prepare(ctx, text)
	if err != nil {
		return QueryResponse{}, err
	}
	if data.Notice != "" {
		return QueryResponse{Answer: "I can help investigate this, but I need an order ID or another identifier to locate the relevant order.", Intent: intent.Intent, OrderID: intent.Entities["order_id"]}, nil
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		return QueryResponse{}, err
	}
	responseCtx, cancel := s.withLLMTimeout(ctx)
	defer cancel()
	answer, err := s.LLM.Complete(responseCtx, llm.CompletionRequest{Messages: []llm.Message{{Role: "system", Content: llm.ResponseSystemPrompt}, {Role: "user", Content: "USER QUERY:\n" + text + "\n\nOPERATIONAL CONTEXT:\n" + string(encoded)}}, JSONMode: false})
	if err != nil {
		return QueryResponse{}, fmt.Errorf("response completion: %w", err)
	}
	return QueryResponse{Answer: strings.TrimSpace(answer), Intent: intent.Intent, OrderID: intent.Entities["order_id"]}, nil
}

func (s *QueryService) Stream(ctx context.Context, text string, callback func(string) error) (query.QueryIntent, error) {
	intent, data, err := s.Prepare(ctx, text)
	if err != nil {
		return intent, err
	}
	if data.Notice != "" {
		return intent, callback("I can help investigate this, but I need an order ID or another identifier to locate the relevant order.")
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		return intent, err
	}
	responseCtx, cancel := s.withLLMTimeout(ctx)
	defer cancel()
	err = s.LLM.Stream(responseCtx, llm.CompletionRequest{Messages: []llm.Message{{Role: "system", Content: llm.ResponseSystemPrompt}, {Role: "user", Content: "USER QUERY:\n" + text + "\n\nOPERATIONAL CONTEXT:\n" + string(encoded)}}}, callback)
	return intent, err
}

var ErrInvalidQuery = errors.New("query cannot be empty")
var ErrQueryTooLong = errors.New("query exceeds maximum length")
var ErrRecordNotFound = errors.New("requested record not found")

func (s *QueryService) withLLMTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if s.LLMTimeout <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, s.LLMTimeout)
}
