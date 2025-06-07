package maxbotapi

import "fmt"

var (
	ErrInvalidChatID    = fmt.Errorf("invalid chat ID")
	ErrEmptyMessage     = fmt.Errorf("message cannot be empty")
	ErrInvalidMessage   = fmt.Errorf("invalid message type")
	ErrRequestFailed    = fmt.Errorf("request failed")
	ErrUnauthorized     = fmt.Errorf("unauthorized")
	ErrRateLimit        = fmt.Errorf("rate limit exceeded")
	ErrWebhookFailed    = fmt.Errorf("webhook processing failed")
	ErrSignatureInvalid = fmt.Errorf("invalid webhook signature")
)

type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e APIError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("API error %d: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("API error %d: %s", e.Code, e.Message)
}
