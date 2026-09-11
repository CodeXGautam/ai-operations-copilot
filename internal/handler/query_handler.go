package handler

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"ai-operations-copilot/internal/service"

	"github.com/gin-gonic/gin"
)

type QueryHandler struct{ Service *service.QueryService }
type request struct {
	Query  string `json:"query"`
	Stream bool   `json:"stream"`
}

func (h *QueryHandler) Query(c *gin.Context) {
	started := time.Now()
	requestID := c.GetString("request_id")
	var input request
	if err := c.ShouldBindJSON(&input); err != nil {
		log.Printf("query_rejected request_id=%s reason=invalid_json duration_ms=%d", requestID, time.Since(started).Milliseconds())
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "request body must be valid JSON")
		return
	}
	if strings.TrimSpace(input.Query) == "" {
		log.Printf("query_rejected request_id=%s reason=empty_query duration_ms=%d", requestID, time.Since(started).Milliseconds())
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "query cannot be empty")
		return
	}
	if input.Stream {
		h.stream(c, input.Query)
		log.Printf("query_completed request_id=%s stream=true duration_ms=%d", requestID, time.Since(started).Milliseconds())
		return
	}
	response, err := h.Service.Process(c.Request.Context(), input.Query)
	if err != nil {
		log.Printf("query_failed request_id=%s error=%q duration_ms=%d", requestID, err, time.Since(started).Milliseconds())
		h.handleError(c, err)
		return
	}
	log.Printf("query_completed request_id=%s intent=%s order_id=%s duration_ms=%d", requestID, response.Intent, response.OrderID, time.Since(started).Milliseconds())
	c.JSON(http.StatusOK, response)
}

func (h *QueryHandler) stream(c *gin.Context, text string) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(http.StatusOK)
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return
	}
	_, err := h.Service.Stream(c.Request.Context(), text, func(token string) error {
		if token == "" {
			return nil
		}
		_, err := c.Writer.WriteString("event: token\ndata: " + strings.ReplaceAll(token, "\n", "\\n") + "\n\n")
		flusher.Flush()
		return err
	})
	if err != nil {
		_, _ = c.Writer.WriteString("event: error\ndata: request failed\n\n")
		flusher.Flush()
		return
	}
	_, _ = c.Writer.WriteString("event: done\ndata: {}\n\n")
	flusher.Flush()
}

func (h *QueryHandler) handleError(c *gin.Context, err error) {
	status, code, message := http.StatusInternalServerError, "INTERNAL_ERROR", "request could not be completed"
	switch {
	case errors.Is(err, service.ErrInvalidQuery), errors.Is(err, service.ErrQueryTooLong):
		status, code, message = http.StatusBadRequest, "INVALID_REQUEST", err.Error()
	case errors.Is(err, service.ErrRecordNotFound):
		status, code, message = http.StatusNotFound, "NOT_FOUND", "the requested operational record was not found"
	case strings.HasPrefix(err.Error(), "invalid intent"):
		status, code, message = http.StatusUnprocessableEntity, "INVALID_INTENT", "the query could not be understood"
	default:
		status, code, message = http.StatusBadGateway, "LLM_PROVIDER_ERROR", "the language model provider could not complete the request"
	}
	writeError(c, status, code, message)
}
func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message}})
}
