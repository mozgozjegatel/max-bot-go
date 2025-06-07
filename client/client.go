package maxbotapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"go.uber.org/zap"
)

const (
	defaultBaseURL = "https://maxbot.yourdomain.com"
	apiVersion     = "v1"
	defaultTimeout = 30 * time.Second
	maxRetries     = 3
	retryDelay     = 1 * time.Second
	rateLimitDelay = 5 * time.Second
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	logger     *zap.Logger
}

type Option func(*Client)

func New(apiKey string, opts ...Option) *Client {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	c := &Client{
		baseURL:    defaultBaseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: defaultTimeout},
		logger:     logger,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = url
	}
}

func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		c.httpClient = client
	}
}

func WithLogger(logger *zap.Logger) Option {
	return func(c *Client) {
		c.logger = logger
	}
}

// doRequest выполняет базовый HTTP запрос к API
func (c *Client) doRequest(
	ctx context.Context,
	method string,
	path string,
	body interface{},
	result interface{},
) error {
	// Подготовка тела запроса
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return fmt.Errorf("encode request body failed: %w", err)
		}
	}

	// Создание HTTP запроса
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, &buf)
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}

	// Установка заголовков
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "max-bot-api-go-client")

	// Выполнение запроса
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Обработка ошибок HTTP
	if resp.StatusCode >= 400 {
		return c.parseAPIError(resp)
	}

	// Парсинг успешного ответа
	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode response failed: %w", err)
		}
	}

	return nil
}

// parseAPIError обрабатывает ошибки API
func (c *Client) parseAPIError(resp *http.Response) error {
	// Чтение тела ошибки с ограничением размера
	const maxErrorSize = 1 << 20 // 1MB
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxErrorSize))
	if err != nil {
		return fmt.Errorf("API error %d (failed to read body: %v)", resp.StatusCode, err)
	}

	// Парсинг стандартной ошибки API
	var apiErr struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Message != "" {
		return fmt.Errorf("API error %d: %s", apiErr.Code, apiErr.Message)
	}

	// Возврат generic ошибки для нестандартных ответов
	return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
}

// GetUpdates получает обновления через long polling
func (c *Client) GetUpdates(ctx context.Context, offset int64) ([]WebhookEvent, error) {
	params := url.Values{}
	params.Set("offset", strconv.FormatInt(offset, 10))

	var updates []WebhookEvent
	err := c.doRequest(ctx, "GET", "/api/v1/updates?"+params.Encode(), nil, &updates)
	if err != nil {
		return nil, fmt.Errorf("get updates failed: %w", err)
	}

	return updates, nil
}

func (c *Client) SendMessage(ctx context.Context, chatID string, message interface{}) (*MessageResponse, error) {
	url := fmt.Sprintf("%s/api/%s/chats/%s/messages", c.baseURL, apiVersion, chatID)

	reqBody, err := json.Marshal(message)
	if err != nil {
		c.logger.Error("Error marshaling message", zap.Error(err))
		return nil, fmt.Errorf("error marshaling message: %w", err)
	}

	var result *MessageResponse
	err = c.retryRequest(ctx, func() error {
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
		if err != nil {
			return fmt.Errorf("error creating request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.apiKey)

		c.logger.Debug("Sending request",
			zap.String("url", url),
			zap.String("method", req.Method),
			zap.Any("body", message),
		)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			c.logger.Warn("Request failed", zap.Error(err))
			return fmt.Errorf("error sending request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			c.logger.Info("Rate limit exceeded, retrying...")
			time.Sleep(rateLimitDelay)
			return ErrRateLimit
		}

		if resp.StatusCode >= 400 {
			return c.parseAPIError(resp)
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("error decoding response: %w", err)
		}

		return nil
	})

	if err != nil {
		c.logger.Error("Request failed after retries", zap.Error(err))
		return nil, err
	}

	c.logger.Info("Message sent successfully", zap.String("chatID", chatID))
	return result, nil
}

// Дополнительные методы API
func (c *Client) GetChat(ctx context.Context, chatID string) (*ChatInfo, error) {
	url := fmt.Sprintf("%s/api/%s/chats/%s", c.baseURL, apiVersion, chatID)

	var chat ChatInfo
	err := c.retryRequest(ctx, func() error {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return fmt.Errorf("error creating request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+c.apiKey)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("error sending request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			return c.parseAPIError(resp)
		}

		return json.NewDecoder(resp.Body).Decode(&chat)
	})

	if err != nil {
		return nil, err
	}

	return &chat, nil
}

func (c *Client) GetMessages(ctx context.Context, chatID string, limit int) ([]Message, error) {
	url := fmt.Sprintf("%s/api/%s/chats/%s/messages?limit=%d", c.baseURL, apiVersion, chatID, limit)

	var messages []Message
	err := c.retryRequest(ctx, func() error {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return fmt.Errorf("error creating request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+c.apiKey)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("error sending request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			return c.parseAPIError(resp)
		}

		return json.NewDecoder(resp.Body).Decode(&messages)
	})

	if err != nil {
		return nil, err
	}

	return messages, nil
}

func (c *Client) StartScenario(ctx context.Context, chatID string, scenarioID string, params map[string]interface{}) (*ScenarioResponse, error) {
	url := fmt.Sprintf("%s/api/%s/chats/%s/scenarios/%s/start", c.baseURL, apiVersion, chatID, scenarioID)

	reqBody, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("error marshaling params: %w", err)
	}

	var result ScenarioResponse
	err = c.retryRequest(ctx, func() error {
		return c.sendRequest(ctx, "POST", url, reqBody, &result)
	})

	return &result, err
}

func (c *Client) StopScenario(ctx context.Context, chatID string, scenarioID string) error {
	url := fmt.Sprintf("%s/api/%s/chats/%s/scenarios/%s/stop", c.baseURL, apiVersion, chatID, scenarioID)
	return c.retryRequest(ctx, func() error {
		return c.sendRequest(ctx, "POST", url, nil, nil)
	})
}

// 2. Методы для работы с сообщениями
func (c *Client) SendKeyboard(ctx context.Context, chatID string, text string, buttons [][]Button) (*MessageResponse, error) {
	msg := struct {
		Text    string     `json:"text"`
		Buttons [][]Button `json:"buttons"`
	}{
		Text:    text,
		Buttons: buttons,
	}

	return c.SendMessage(ctx, chatID, msg)
}

func (c *Client) SendCarousel(ctx context.Context, chatID string, items []CarouselItem) (*MessageResponse, error) {
	return c.SendMessage(ctx, chatID, map[string]interface{}{
		"carousel": items,
	})
}

// 3. Методы управления чатами
func (c *Client) SetChatVariables(ctx context.Context, chatID string, variables map[string]interface{}) error {
	url := fmt.Sprintf("%s/api/%s/chats/%s/variables", c.baseURL, apiVersion, chatID)
	return c.retryRequest(ctx, func() error {
		return c.sendRequest(ctx, "PUT", url, variables, nil)
	})
}

func (c *Client) TransferToAgent(ctx context.Context, chatID string, options TransferOptions) error {
	url := fmt.Sprintf("%s/api/%s/chats/%s/transfer", c.baseURL, apiVersion, chatID)
	return c.retryRequest(ctx, func() error {
		return c.sendRequest(ctx, "POST", url, options, nil)
	})
}

// 4. Вспомогательные методы
func (c *Client) sendRequest(ctx context.Context, method string, url string, body interface{}, result interface{}) error {
	var reqBody []byte
	if body != nil {
		var err error
		reqBody, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("error marshaling body: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return c.parseAPIError(resp)
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("error decoding response: %w", err)
		}
	}

	return nil
}
