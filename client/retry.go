package maxbotapi

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
)

func (c *Client) retryRequest(ctx context.Context, fn func() error) error {
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Не повторяем для некоторых ошибок
		if errors.Is(err, ErrInvalidChatID) || errors.Is(err, ErrUnauthorized) {
			break
		}

		c.logger.Info("Retrying request",
			zap.Int("attempt", i+1),
			zap.Error(err),
		)

		if i < maxRetries-1 {
			time.Sleep(retryDelay)
		}
	}

	return lastErr
}
