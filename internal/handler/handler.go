package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"green-api-client/internal/greenapi"
)

// Handler — HTTP-хендлеры для проксирования запросов к GREEN-API.
type Handler struct {
	log             *slog.Logger
	greenAPIBaseURL string
}

// Option — функциональная опция для Handler
type Option func(*Handler)

// WithGreenAPIBaseURL устанавливает кастомный базовый URL GREEN-API (для тестирования)
func WithGreenAPIBaseURL(url string) Option {
	return func(h *Handler) {
		h.greenAPIBaseURL = url
	}
}

// New создаёт новый Handler.
func New(log *slog.Logger, opts ...Option) *Handler {
	h := &Handler{log: log}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// newClient создаёт greenapi.Client с учётом переопределённого base URL
func (h *Handler) newClient(idInstance, apiToken string) *greenapi.Client {
	var opts []greenapi.Option
	if h.greenAPIBaseURL != "" {
		opts = append(opts, greenapi.WithBaseURL(h.greenAPIBaseURL))
	}
	return greenapi.NewClient(idInstance, apiToken, opts...)
}

// credentials — общие поля авторизации, присутствующие в каждом запросе.
type credentials struct {
	IDInstance       string `json:"idInstance"`
	APITokenInstance string `json:"apiTokenInstance"`
}

// errorResponse — формат ответа при ошибке.
type errorResponse struct {
	Error string `json:"error"`
}

// GetSettings обрабатывает POST /api/getSettings.
func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	const op = "handler.GetSettings"

	var req credentials
	if !h.decodeJSON(w, r, op, &req) {
		return
	}

	if !h.validateCredentials(w, op, req) {
		return
	}

	client := h.newClient(req.IDInstance, req.APITokenInstance)

	data, err := client.GetSettings(r.Context())
	if err != nil {
		h.writeError(w, op, http.StatusBadGateway, err)
		return
	}

	h.log.Info("settings retrieved", slog.String("op", op))
	h.writeJSON(w, http.StatusOK, data)
}

// GetStateInstance обрабатывает POST /api/getStateInstance.
func (h *Handler) GetStateInstance(w http.ResponseWriter, r *http.Request) {
	const op = "handler.GetStateInstance"

	var req credentials
	if !h.decodeJSON(w, r, op, &req) {
		return
	}

	if !h.validateCredentials(w, op, req) {
		return
	}

	client := h.newClient(req.IDInstance, req.APITokenInstance)

	data, err := client.GetStateInstance(r.Context())
	if err != nil {
		h.writeError(w, op, http.StatusBadGateway, err)
		return
	}

	h.log.Info("state retrieved", slog.String("op", op))
	h.writeJSON(w, http.StatusOK, data)
}

// sendMessageRequest — тело запроса для sendMessage.
type sendMessageRequest struct {
	credentials
	ChatID  string `json:"chatId"`
	Message string `json:"message"`
}

// SendMessage обрабатывает POST /api/sendMessage.
func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	const op = "handler.SendMessage"

	var req sendMessageRequest
	if !h.decodeJSON(w, r, op, &req) {
		return
	}

	if !h.validateCredentials(w, op, req.credentials) {
		return
	}

	if req.ChatID == "" || req.Message == "" {
		h.writeJSON(w, http.StatusBadRequest, errorResponse{Error: "chatId and message are required"})
		return
	}

	client := h.newClient(req.IDInstance, req.APITokenInstance)

	data, err := client.SendMessage(r.Context(), req.ChatID, req.Message)
	if err != nil {
		h.writeError(w, op, http.StatusBadGateway, err)
		return
	}

	h.log.Info("message sent", slog.String("op", op), slog.String("chatId", req.ChatID))
	h.writeJSON(w, http.StatusOK, data)
}

// sendFileByURLRequest — тело запроса для sendFileByUrl.
type sendFileByURLRequest struct {
	credentials
	ChatID   string `json:"chatId"`
	URLFile  string `json:"urlFile"`
	FileName string `json:"fileName"`
}

// SendFileByURL обрабатывает POST /api/sendFileByUrl.
func (h *Handler) SendFileByURL(w http.ResponseWriter, r *http.Request) {
	const op = "handler.SendFileByURL"

	var req sendFileByURLRequest
	if !h.decodeJSON(w, r, op, &req) {
		return
	}

	if !h.validateCredentials(w, op, req.credentials) {
		return
	}

	if req.ChatID == "" || req.URLFile == "" || req.FileName == "" {
		h.writeJSON(w, http.StatusBadRequest, errorResponse{Error: "chatId, urlFile and fileName are required"})
		return
	}

	client := h.newClient(req.IDInstance, req.APITokenInstance)

	data, err := client.SendFileByURL(r.Context(), req.ChatID, req.URLFile, req.FileName)
	if err != nil {
		h.writeError(w, op, http.StatusBadGateway, err)
		return
	}

	h.log.Info("file sent", slog.String("op", op), slog.String("chatId", req.ChatID), slog.String("fileName", req.FileName))
	h.writeJSON(w, http.StatusOK, data)
}

// decodeJSON декодирует JSON из тела запроса. Возвращает false при ошибке (ответ уже записан).
func (h *Handler) decodeJSON(w http.ResponseWriter, r *http.Request, op string, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		h.log.Warn("invalid request body", slog.String("op", op), slog.String("error", err.Error()))
		h.writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return false
	}
	return true
}

// validateCredentials проверяет наличие idInstance и apiTokenInstance.
func (h *Handler) validateCredentials(w http.ResponseWriter, op string, creds credentials) bool {
	if creds.IDInstance == "" || creds.APITokenInstance == "" {
		h.log.Warn("missing credentials", slog.String("op", op))
		h.writeJSON(w, http.StatusBadRequest, errorResponse{Error: "idInstance and apiTokenInstance are required"})
		return false
	}
	return true
}

// writeError логирует ошибку и отправляет JSON-ответ с сообщением.
func (h *Handler) writeError(w http.ResponseWriter, op string, status int, err error) {
	h.log.Error("upstream error", slog.String("op", op), slog.String("error", err.Error()))
	h.writeJSON(w, status, errorResponse{Error: err.Error()})
}

// writeJSON сериализует v в JSON и записывает в ResponseWriter.
func (h *Handler) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
