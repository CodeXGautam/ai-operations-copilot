package llm

import "context"

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type CompletionRequest struct {
	Messages []Message
	JSONMode bool
}
type Client interface {
	Complete(context.Context, CompletionRequest) (string, error)
	Stream(context.Context, CompletionRequest, func(string) error) error
}
