package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// greenAPIStub возвращает httptest.Server, имитирующий GREEN-API
func greenAPIStub(t *testing.T, statusCode int, response string) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(ts.Close)
	return ts
}

func newTestHandler(t *testing.T, stubURL string) *Handler {
	t.Helper()
	return New(slog.Default(), WithGreenAPIBaseURL(stubURL))
}

func postJSON(target string, body string) *http.Request {
	return httptest.NewRequestWithContext(context.Background(), http.MethodPost, target, strings.NewReader(body))
}

func TestGetSettings(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		stubStatus   int
		stubResponse string
		wantStatus   int
		wantErrField bool
	}{
		{
			name:         "success",
			body:         `{"idInstance":"123","apiTokenInstance":"abc"}`,
			stubStatus:   http.StatusOK,
			stubResponse: `{"wid":"79001234567@c.us"}`,
			wantStatus:   http.StatusOK,
		},
		{
			name:         "empty body",
			body:         "",
			wantStatus:   http.StatusBadRequest,
			wantErrField: true,
		},
		{
			name:         "invalid json",
			body:         "not-json",
			wantStatus:   http.StatusBadRequest,
			wantErrField: true,
		},
		{
			name:         "missing credentials",
			body:         `{"idInstance":"","apiTokenInstance":""}`,
			wantStatus:   http.StatusBadRequest,
			wantErrField: true,
		},
		{
			name:         "missing apiTokenInstance",
			body:         `{"idInstance":"123"}`,
			wantStatus:   http.StatusBadRequest,
			wantErrField: true,
		},
		{
			name:         "upstream error",
			body:         `{"idInstance":"123","apiTokenInstance":"abc"}`,
			stubStatus:   http.StatusInternalServerError,
			stubResponse: `{"error":"fail"}`,
			wantStatus:   http.StatusBadGateway,
			wantErrField: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := greenAPIStub(t, tt.stubStatus, tt.stubResponse)
			h := newTestHandler(t, stub.URL)

			rec := httptest.NewRecorder()
			h.GetSettings(rec, postJSON("/api/getSettings", tt.body))

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

			if tt.wantErrField {
				var resp errorResponse
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
				assert.NotEmpty(t, resp.Error)
			}
		})
	}
}

func TestGetStateInstance(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		stubStatus   int
		stubResponse string
		wantStatus   int
		wantErrField bool
	}{
		{
			name:         "success",
			body:         `{"idInstance":"123","apiTokenInstance":"abc"}`,
			stubStatus:   http.StatusOK,
			stubResponse: `{"stateInstance":"authorized"}`,
			wantStatus:   http.StatusOK,
		},
		{
			name:         "empty body",
			body:         "",
			wantStatus:   http.StatusBadRequest,
			wantErrField: true,
		},
		{
			name:         "missing credentials",
			body:         `{"idInstance":"123","apiTokenInstance":""}`,
			wantStatus:   http.StatusBadRequest,
			wantErrField: true,
		},
		{
			name:         "upstream error",
			body:         `{"idInstance":"123","apiTokenInstance":"abc"}`,
			stubStatus:   http.StatusBadGateway,
			stubResponse: `bad`,
			wantStatus:   http.StatusBadGateway,
			wantErrField: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := greenAPIStub(t, tt.stubStatus, tt.stubResponse)
			h := newTestHandler(t, stub.URL)

			rec := httptest.NewRecorder()
			h.GetStateInstance(rec, postJSON("/api/getStateInstance", tt.body))

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

			if tt.wantErrField {
				var resp errorResponse
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
				assert.NotEmpty(t, resp.Error)
			}
		})
	}
}

func TestSendMessage(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		stubStatus   int
		stubResponse string
		wantStatus   int
		wantErrField bool
	}{
		{
			name:         "success",
			body:         `{"idInstance":"123","apiTokenInstance":"abc","chatId":"79001234567@c.us","message":"Hello!"}`,
			stubStatus:   http.StatusOK,
			stubResponse: `{"idMessage":"3EB0C767D72"}`,
			wantStatus:   http.StatusOK,
		},
		{
			name:         "empty body",
			body:         "",
			wantStatus:   http.StatusBadRequest,
			wantErrField: true,
		},
		{
			name:         "missing credentials",
			body:         `{"chatId":"79001234567@c.us","message":"Hello!"}`,
			wantStatus:   http.StatusBadRequest,
			wantErrField: true,
		},
		{
			name:         "missing chatId",
			body:         `{"idInstance":"123","apiTokenInstance":"abc","message":"Hello!"}`,
			wantStatus:   http.StatusBadRequest,
			wantErrField: true,
		},
		{
			name:         "missing message",
			body:         `{"idInstance":"123","apiTokenInstance":"abc","chatId":"79001234567@c.us"}`,
			wantStatus:   http.StatusBadRequest,
			wantErrField: true,
		},
		{
			name:         "both chatId and message missing",
			body:         `{"idInstance":"123","apiTokenInstance":"abc"}`,
			wantStatus:   http.StatusBadRequest,
			wantErrField: true,
		},
		{
			name:         "upstream error",
			body:         `{"idInstance":"123","apiTokenInstance":"abc","chatId":"79001234567@c.us","message":"Hello!"}`,
			stubStatus:   http.StatusInternalServerError,
			stubResponse: `error`,
			wantStatus:   http.StatusBadGateway,
			wantErrField: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := greenAPIStub(t, tt.stubStatus, tt.stubResponse)
			h := newTestHandler(t, stub.URL)

			rec := httptest.NewRecorder()
			h.SendMessage(rec, postJSON("/api/sendMessage", tt.body))

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

			if tt.wantErrField {
				var resp errorResponse
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
				assert.NotEmpty(t, resp.Error)
			}
		})
	}
}

func TestSendFileByURL(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		stubStatus   int
		stubResponse string
		wantStatus   int
		wantErrField bool
	}{
		{
			name:         "success",
			body:         `{"idInstance":"123","apiTokenInstance":"abc","chatId":"79001234567@c.us","urlFile":"https://example.com/img.png","fileName":"img.png"}`,
			stubStatus:   http.StatusOK,
			stubResponse: `{"idMessage":"3EB0C767D72"}`,
			wantStatus:   http.StatusOK,
		},
		{
			name:         "empty body",
			body:         "",
			wantStatus:   http.StatusBadRequest,
			wantErrField: true,
		},
		{
			name:         "missing credentials",
			body:         `{"chatId":"79001234567@c.us","urlFile":"https://example.com/img.png","fileName":"img.png"}`,
			wantStatus:   http.StatusBadRequest,
			wantErrField: true,
		},
		{
			name:         "missing chatId",
			body:         `{"idInstance":"123","apiTokenInstance":"abc","urlFile":"https://example.com/img.png","fileName":"img.png"}`,
			wantStatus:   http.StatusBadRequest,
			wantErrField: true,
		},
		{
			name:         "missing urlFile",
			body:         `{"idInstance":"123","apiTokenInstance":"abc","chatId":"79001234567@c.us","fileName":"img.png"}`,
			wantStatus:   http.StatusBadRequest,
			wantErrField: true,
		},
		{
			name:         "missing fileName",
			body:         `{"idInstance":"123","apiTokenInstance":"abc","chatId":"79001234567@c.us","urlFile":"https://example.com/img.png"}`,
			wantStatus:   http.StatusBadRequest,
			wantErrField: true,
		},
		{
			name:         "all fields missing",
			body:         `{"idInstance":"123","apiTokenInstance":"abc"}`,
			wantStatus:   http.StatusBadRequest,
			wantErrField: true,
		},
		{
			name:         "upstream error",
			body:         `{"idInstance":"123","apiTokenInstance":"abc","chatId":"79001234567@c.us","urlFile":"https://example.com/img.png","fileName":"img.png"}`,
			stubStatus:   http.StatusForbidden,
			stubResponse: `forbidden`,
			wantStatus:   http.StatusBadGateway,
			wantErrField: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := greenAPIStub(t, tt.stubStatus, tt.stubResponse)
			h := newTestHandler(t, stub.URL)

			rec := httptest.NewRecorder()
			h.SendFileByURL(rec, postJSON("/api/sendFileByUrl", tt.body))

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

			if tt.wantErrField {
				var resp errorResponse
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
				assert.NotEmpty(t, resp.Error)
			}
		})
	}
}

func TestSuccessResponseBody(t *testing.T) {
	stub := greenAPIStub(t, http.StatusOK, `{"wid":"79001234567@c.us"}`)
	h := newTestHandler(t, stub.URL)

	rec := httptest.NewRecorder()
	h.GetSettings(rec, postJSON("/api/getSettings", `{"idInstance":"123","apiTokenInstance":"abc"}`))

	require.Equal(t, http.StatusOK, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "79001234567@c.us", body["wid"])
}
