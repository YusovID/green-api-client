package greenapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	return ts
}

func newTestClient(t *testing.T, serverURL string) *Client {
	t.Helper()
	c := NewClient("testID", "testToken")
	c.baseURL = serverURL
	return c
}

func TestClient_URL(t *testing.T) {
	c := NewClient("123", "abc")
	got := c.url("getSettings")
	assert.Equal(t, "https://api.green-api.com/waInstance123/getSettings/abc", got)
}

func TestClient_GetSettings(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   string
		wantErr    bool
	}{
		{
			name:       "success",
			statusCode: http.StatusOK,
			response:   `{"wid":"79001234567@c.us","webhookUrl":""}`,
		},
		{
			name:       "bad request",
			statusCode: http.StatusBadRequest,
			response:   `{"error":"bad request"}`,
			wantErr:    true,
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			response:   `{"error":"internal"}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Contains(t, r.URL.Path, "/waInstancetestID/getSettings/testToken")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.response))
			})

			client := newTestClient(t, ts.URL)
			data, err := client.GetSettings(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, data)
			} else {
				require.NoError(t, err)
				assert.JSONEq(t, tt.response, string(data))
			}
		})
	}
}

func TestClient_GetStateInstance(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   string
		wantErr    bool
	}{
		{
			name:       "success",
			statusCode: http.StatusOK,
			response:   `{"stateInstance":"authorized"}`,
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			response:   `{"error":"unauthorized"}`,
			wantErr:    true,
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			response:   `internal error`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Contains(t, r.URL.Path, "/waInstancetestID/getStateInstance/testToken")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.response))
			})

			client := newTestClient(t, ts.URL)
			data, err := client.GetStateInstance(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, data)
			} else {
				require.NoError(t, err)
				assert.JSONEq(t, tt.response, string(data))
			}
		})
	}
}

func TestClient_SendMessage(t *testing.T) {
	tests := []struct {
		name       string
		chatID     string
		message    string
		statusCode int
		response   string
		wantErr    bool
	}{
		{
			name:       "success",
			chatID:     "79001234567@c.us",
			message:    "Hello!",
			statusCode: http.StatusOK,
			response:   `{"idMessage":"3EB0C767D72"}`,
		},
		{
			name:       "bad request",
			chatID:     "invalid",
			message:    "test",
			statusCode: http.StatusBadRequest,
			response:   `{"error":"invalid chatId"}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Contains(t, r.URL.Path, "/waInstancetestID/sendMessage/testToken")
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)

				var req sendMessageRequest
				require.NoError(t, json.Unmarshal(body, &req))
				assert.Equal(t, tt.chatID, req.ChatID)
				assert.Equal(t, tt.message, req.Message)

				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.response))
			})

			client := newTestClient(t, ts.URL)
			data, err := client.SendMessage(context.Background(), tt.chatID, tt.message)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, data)
			} else {
				require.NoError(t, err)
				assert.JSONEq(t, tt.response, string(data))
			}
		})
	}
}

func TestClient_SendFileByURL(t *testing.T) {
	tests := []struct {
		name       string
		chatID     string
		urlFile    string
		fileName   string
		statusCode int
		response   string
		wantErr    bool
	}{
		{
			name:       "success",
			chatID:     "79001234567@c.us",
			urlFile:    "https://example.com/image.png",
			fileName:   "image.png",
			statusCode: http.StatusOK,
			response:   `{"idMessage":"3EB0C767D72","urlFile":"https://example.com/image.png"}`,
		},
		{
			name:       "forbidden",
			chatID:     "79001234567@c.us",
			urlFile:    "https://example.com/file.pdf",
			fileName:   "file.pdf",
			statusCode: http.StatusForbidden,
			response:   `{"error":"forbidden"}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Contains(t, r.URL.Path, "/waInstancetestID/sendFileByUrl/testToken")
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)

				var req sendFileByURLRequest
				require.NoError(t, json.Unmarshal(body, &req))
				assert.Equal(t, tt.chatID, req.ChatID)
				assert.Equal(t, tt.urlFile, req.URLFile)
				assert.Equal(t, tt.fileName, req.FileName)

				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.response))
			})

			client := newTestClient(t, ts.URL)
			data, err := client.SendFileByURL(context.Background(), tt.chatID, tt.urlFile, tt.fileName)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, data)
			} else {
				require.NoError(t, err)
				assert.JSONEq(t, tt.response, string(data))
			}
		})
	}
}

func TestClient_ContextCancelled(t *testing.T) {
	ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	client := newTestClient(t, ts.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.GetSettings(ctx)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestClient_ConnectionError(t *testing.T) {
	client := NewClient("id", "token", WithBaseURL("http://127.0.0.1:1"))

	_, err := client.GetSettings(context.Background())
	require.Error(t, err)
	assert.NotErrorIs(t, err, context.Canceled)
	assert.NotErrorIs(t, err, context.DeadlineExceeded)
}

func TestClient_ContextDeadlineExceeded(t *testing.T) {
	ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	client := newTestClient(t, ts.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := client.GetSettings(ctx)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestClient_EmptyResponseBody(t *testing.T) {
	ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	client := newTestClient(t, ts.URL)
	data, err := client.GetSettings(context.Background())

	require.NoError(t, err)
	assert.Empty(t, data)
}

func TestClient_WithBaseURL(t *testing.T) {
	ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	client := NewClient("id", "token", WithBaseURL(ts.URL))
	data, err := client.GetSettings(context.Background())

	require.NoError(t, err)
	assert.JSONEq(t, `{"ok":true}`, string(data))
}
