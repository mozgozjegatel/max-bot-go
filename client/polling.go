// client/polling.go
package maxbotapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"go.uber.org/zap"
)

type PollingConfig struct {
	Timeout      time.Duration
	RetryDelay   time.Duration
	BufferSize   int
	UpdateOffset int64
}

type PollingUpdate struct {
	UpdateID int64
	Event    *WebhookEvent
	Error    error
}

func DefaultPollingConfig() *PollingConfig {
	return &PollingConfig{
		Timeout:      25 * time.Second,
		RetryDelay:   1 * time.Second,
		BufferSize:   100,
		UpdateOffset: 0,
	}
}

func (c *Client) StartPolling(ctx context.Context, config *PollingConfig) <-chan PollingUpdate {
	if config == nil {
		config = DefaultPollingConfig()
	}

	updates := make(chan PollingUpdate, config.BufferSize)

	go c.pollingWorker(ctx, config, updates)
	return updates
}

func (c *Client) pollingWorker(ctx context.Context, config *PollingConfig, updates chan<- PollingUpdate) {
	defer close(updates)

	params := url.Values{}
	params.Set("timeout", strconv.Itoa(int(config.Timeout.Seconds())))

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Polling stopped by context")
			return
		default:
			params.Set("offset", strconv.FormatInt(config.UpdateOffset, 10))
			url := fmt.Sprintf("%s/api/%s/getUpdates?%s", c.baseURL, apiVersion, params.Encode())

			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				c.logger.Error("Error creating polling request", zap.Error(err))
				sendUpdateError(updates, err)
				time.Sleep(config.RetryDelay)
				continue
			}

			req.Header.Set("Authorization", "Bearer "+c.apiKey)

			resp, err := c.httpClient.Do(req)
			if err != nil {
				c.logger.Warn("Polling request failed", zap.Error(err))
				sendUpdateError(updates, err)
				time.Sleep(config.RetryDelay)
				continue
			}

			if resp.StatusCode != http.StatusOK {
				err := c.parseAPIError(resp)
				c.logger.Warn("Polling request failed", zap.Error(err))
				sendUpdateError(updates, err)
				resp.Body.Close()
				time.Sleep(config.RetryDelay)
				continue
			}

			var apiResponse struct {
				OK     bool            `json:"ok"`
				Result []*WebhookEvent `json:"result"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
				c.logger.Error("Error decoding polling response", zap.Error(err))
				sendUpdateError(updates, err)
				resp.Body.Close()
				time.Sleep(config.RetryDelay)
				continue
			}
			resp.Body.Close()

			if !apiResponse.OK {
				c.logger.Warn("Polling response not OK")
				time.Sleep(config.RetryDelay)
				continue
			}

			for _, update := range apiResponse.Result {
				select {
				case updates <- PollingUpdate{
					UpdateID: update.UpdateID,
					Event:    update,
				}:
					config.UpdateOffset = update.UpdateID + 1
				case <-ctx.Done():
					c.logger.Info("Polling stopped by context during updates processing")
					return
				}
			}
		}
	}
}

func sendUpdateError(updates chan<- PollingUpdate, err error) {
	select {
	case updates <- PollingUpdate{Error: err}:
	default:
		// Не блокируем, если канал полон
	}
}
