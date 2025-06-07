package maxbotapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"go.uber.org/zap"
)

type WebhookHandler struct {
	secret string
	logger *zap.Logger
}

func NewWebhookHandler(secret string, logger *zap.Logger) *WebhookHandler {
	if logger == nil {
		var err error
		logger, err = zap.NewProduction()
		if err != nil {
			// Fallback to basic logger if production logger fails
			logger = zap.NewExample()
		}
	}

	return &WebhookHandler{
		secret: secret,
		logger: logger,
	}
}

func (wh *WebhookHandler) VerifySignature(signature string, body []byte) bool {
	if wh.secret == "" {
		wh.logger.Warn("Webhook secret not set, skipping signature verification")
		return true
	}

	mac := hmac.New(sha256.New, []byte(wh.secret))
	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	wh.logger.Debug("Signature verification",
		zap.String("received", signature),
		zap.String("expected", expectedSignature),
	)

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

func (wh *WebhookHandler) ParseRequest(r *http.Request) (*WebhookEvent, error) {
	if r.Method != http.MethodPost {
		return nil, errors.New("invalid HTTP method, expected POST")
	}

	signature := r.Header.Get("X-Signature")
	if signature == "" {
		return nil, errors.New("missing X-Signature header")
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading body: %w", err)
	}
	defer r.Body.Close()

	if !wh.VerifySignature(signature, body) {
		return nil, ErrSignatureInvalid
	}

	var event WebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, fmt.Errorf("error unmarshaling event: %w", err)
	}

	if event.Type == "" {
		return nil, errors.New("missing event type")
	}

	wh.logger.Info("Webhook event received",
		zap.String("type", event.Type),
		zap.String("chatID", event.Chat.ID),
	)

	return &event, nil
}

func (wh *WebhookHandler) Handle(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		event, err := wh.ParseRequest(r)
		if err != nil {
			wh.logger.Error("Webhook error", zap.Error(err))
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(r.Context(), "webhookEvent", event)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
