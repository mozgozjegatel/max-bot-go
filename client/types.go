package maxbotapi

import (
	"encoding/json"
	"time"
)

type MessageResponse struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Status    string    `json:"status"`
}

type ChatInfo struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Status    string    `json:"status"`
	User      User      `json:"user"`
}

type User struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
}

type Message struct {
	ID        string          `json:"id"`
	ChatID    string          `json:"chat_id"`
	Text      string          `json:"text"`
	Direction string          `json:"direction"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

type WebhookEvent struct {
	UpdateID  int64           `json:"update_id"`  // Обязательное поле, соответствует TS
	EventID   string          `json:"event_id"`   // Уникальный ID события
	Type      string          `json:"type"`       // Тип события: "message", "button", etc.
	Chat      Chat            `json:"chat"`       // Информация о чате
	Message   *Message        `json:"message"`    // Сообщение (для message events)
	User      *User           `json:"user"`       // Пользователь
	Data      json.RawMessage `json:"data"`       // Дополнительные данные
	CreatedAt time.Time       `json:"created_at"` // Временная метка
}

type EventData struct {
	// Добавьте необходимые поля данных события
}

type Chat struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Status    string            `json:"status"`
	User      User              `json:"user"`
	Variables map[string]string `json:"variables"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

type TextMessage struct {
	Text string `json:"text"`
}

type ImageMessage struct {
	ImageURL string `json:"image_url"`
}

// type Button struct {
// 	Text string `json:"text"`
// 	URL  string `json:"url"`
// }

type ButtonsMessage struct {
	Text    string   `json:"text"`
	Buttons []Button `json:"buttons"`
}

// Структуры для работы со сценариями
type ScenarioResponse struct {
	SessionID string    `json:"session_id"`
	Status    string    `json:"status"`
	StartedAt time.Time `json:"started_at"`
}

type ScenarioStep struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Payload   map[string]interface{} `json:"payload"`
	CreatedAt time.Time              `json:"created_at"`
}

// Структуры для клавиатур и каруселей
type Button struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Type  string `json:"type"` // "text", "url", etc.
	Value string `json:"value"`
}

type CarouselItem struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	ImageURL    string   `json:"image_url"`
	Buttons     []Button `json:"buttons"`
}

// Структуры для управления чатами
type ChatVariables struct {
	Variables map[string]interface{} `json:"variables"`
}

type TransferOptions struct {
	AgentID  string            `json:"agent_id,omitempty"`
	GroupID  string            `json:"group_id,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Дополнительные типы сообщений
type LocationMessage struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Title     string  `json:"title,omitempty"`
}

type ContactMessage struct {
	PhoneNumber string `json:"phone_number"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name,omitempty"`
}

type TemplateMessage struct {
	TemplateID string                 `json:"template_id"`
	Variables  map[string]interface{} `json:"variables,omitempty"`
}

// Chat соответствует интерфейсу IChat из TS
// type Chat struct {
// 	ID        string            `json:"id"`
// 	Type      string            `json:"type"`
// 	Status    string            `json:"status"`
// 	User      User              `json:"user"`
// 	Variables map[string]string `json:"variables"`
// 	CreatedAt time.Time         `json:"created_at"`
// 	UpdatedAt time.Time         `json:"updated_at"`
// }

// Message соответствует IMessage из TS
// type Message struct {
// 	ID        string          `json:"id"`
// 	ChatID    string          `json:"chat_id"`
// 	Text      string          `json:"text"`
// 	Direction string          `json:"direction"`
// 	Type      string          `json:"type"`
// 	Payload   json.RawMessage `json:"payload"`
// 	CreatedAt time.Time       `json:"created_at"`
// }

// ScenarioSession соответствует IScenarioSession из TS
type ScenarioSession struct {
	ID          string            `json:"id"`
	Scenario    Scenario          `json:"scenario"`
	Chat        Chat              `json:"chat"`
	State       map[string]string `json:"state"`
	CurrentStep StepExecution     `json:"current_step"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// WebhookEvent соответствует IWebhookEvent из TS
// type WebhookEvent struct {
// 	EventID   string          `json:"event_id"`
// 	Type      string          `json:"type"`
// 	Chat      Chat            `json:"chat"`
// 	Message   *Message        `json:"message"`
// 	Data      json.RawMessage `json:"data"`
// 	CreatedAt time.Time       `json:"created_at"`
// }

// Scenario представляет сценарий бота (соответствует IScenario из TS-клиента)
type Scenario struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Version     string              `json:"version"`
	Steps       map[string]Step     `json:"steps"`
	Variables   map[string]Variable `json:"variables"`
	Settings    ScenarioSettings    `json:"settings"`
	Metadata    json.RawMessage     `json:"metadata"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

// Step представляет шаг сценария
type Step struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"` // "message", "input", "condition", etc.
	Payload   json.RawMessage `json:"payload"`
	NextSteps []NextStep      `json:"next_steps"`
	Timeout   int             `json:"timeout"`
	ErrorStep string          `json:"error_step"`
}

// NextStep определяет переход между шагами
type NextStep struct {
	Condition string `json:"condition"`
	StepID    string `json:"step_id"`
}

// Variable определяет переменную сценария
type Variable struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // "string", "number", "boolean", etc.
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Default     string `json:"default"`
}

// ScenarioSettings содержит настройки сценария
type ScenarioSettings struct {
	Timeout           int  `json:"timeout"`
	AllowInterruption bool `json:"allow_interruption"`
	Restartable       bool `json:"restartable"`
}

// StepExecution содержит информацию о выполняемом шаге
type StepExecution struct {
	StepID      string          `json:"step_id"`
	StartedAt   time.Time       `json:"started_at"`
	CompletedAt time.Time       `json:"completed_at"`
	Payload     json.RawMessage `json:"payload"`
}
