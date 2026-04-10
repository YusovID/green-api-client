package greenapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultBaseURL = "https://api.green-api.com"
	httpTimeout    = 15 * time.Second
)

// Client — HTTP-клиент для работы с GREEN-API
type Client struct {
	httpClient *http.Client
	baseURL    string
	idInstance string
	apiToken   string
}

// Option — функциональная опция для Client
type Option func(*Client)

// WithBaseURL устанавливает кастомный базовый URL
func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = url
	}
}

// NewClient создаёт новый клиент GREEN-API
func NewClient(idInstance, apiToken string, opts ...Option) *Client {
	c := &Client{
		httpClient: &http.Client{Timeout: httpTimeout},
		baseURL:    defaultBaseURL,
		idInstance: idInstance,
		apiToken:   apiToken,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// GetSettings возвращает настройки инстанса
func (c *Client) GetSettings(ctx context.Context) (json.RawMessage, error) {
	const op = "greenapi.Client.GetSettings"

	return c.doGet(ctx, op, "getSettings")
}

// GetStateInstance возвращает состояние инстанса
func (c *Client) GetStateInstance(ctx context.Context) (json.RawMessage, error) {
	const op = "greenapi.Client.GetStateInstance"

	return c.doGet(ctx, op, "getStateInstance")
}

// sendMessageRequest — тело запроса для sendMessage
type sendMessageRequest struct {
	ChatID  string `json:"chatId"`
	Message string `json:"message"`
}

// SendMessage отправляет текстовое сообщение
func (c *Client) SendMessage(ctx context.Context, chatID, message string) (json.RawMessage, error) {
	const op = "greenapi.Client.SendMessage"

	body := sendMessageRequest{
		ChatID:  chatID,
		Message: message,
	}

	return c.doPost(ctx, op, "sendMessage", body)
}

// sendFileByURLRequest — тело запроса для sendFileByUrl
type sendFileByURLRequest struct {
	ChatID   string `json:"chatId"`
	URLFile  string `json:"urlFile"`
	FileName string `json:"fileName"`
}

// SendFileByURL отправляет файл по ссылке
func (c *Client) SendFileByURL(ctx context.Context, chatID, urlFile, fileName string) (json.RawMessage, error) {
	const op = "greenapi.Client.SendFileByURL"

	body := sendFileByURLRequest{
		ChatID:   chatID,
		URLFile:  urlFile,
		FileName: fileName,
	}

	return c.doPost(ctx, op, "sendFileByUrl", body)
}

// url формирует URL для запроса к GREEN-API
func (c *Client) url(method string) string {
	return fmt.Sprintf("%s/waInstance%s/%s/%s", c.baseURL, c.idInstance, method, c.apiToken)
}

// doGet выполняет GET-запрос к GREEN-API
func (c *Client) doGet(ctx context.Context, op, method string) (json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url(method), nil)
	if err != nil {
		return nil, fmt.Errorf("%s: create request: %w", op, err)
	}

	return c.do(req, op)
}

// doPost выполняет POST-запрос к GREEN-API с JSON-телом
func (c *Client) doPost(ctx context.Context, op, method string, body any) (json.RawMessage, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("%s: marshal body: %w", op, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url(method), bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%s: create request: %w", op, err)
	}

	req.Header.Set("Content-Type", "application/json")

	return c.do(req, op)
}

// do выполняет HTTP-запрос и возвращает тело ответа
func (c *Client) do(req *http.Request, op string) (json.RawMessage, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: do request: %w", op, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%s: read response: %w", op, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s: unexpected status %d: %s", op, resp.StatusCode, string(respBody))
	}

	return json.RawMessage(respBody), nil
}
