package telegramdelivery_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/telegramdelivery"
)

func TestTelegramAdapterSendsTextMessage(t *testing.T) {
	var requestBody map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bottoken/sendMessage" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":123}}`))
	}))
	defer server.Close()
	adapter := newTestAdapter(t, server.URL)

	result, err := adapter.DispatchDeliveryStep(context.Background(), model.DeliveryDispatchStep{
		StepIndex: 1,
		Kind:      model.DeliveryDispatchStepText,
		Channel:   "telegram",
		ChatID:    "8655199155",
		Message:   "hello",
	})
	if err != nil {
		t.Fatalf("dispatch text: %v", err)
	}
	if requestBody["chat_id"] != "8655199155" || requestBody["text"] != "hello" {
		t.Fatalf("unexpected request body: %+v", requestBody)
	}
	if result.Provider != "telegram" || result.ProviderMessageID != "123" || result.Status != model.DeliveryDispatchSent {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestTelegramAdapterSendsLocalPhotoMultipart(t *testing.T) {
	dir := t.TempDir()
	imagePath := filepath.Join(dir, "image.png")
	if err := os.WriteFile(imagePath, []byte("image-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bottoken/sendPhoto" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Fatalf("expected multipart content type, got %s", r.Header.Get("Content-Type"))
		}
		if err := r.ParseMultipartForm(1024 * 1024); err != nil {
			t.Fatal(err)
		}
		if got := r.FormValue("chat_id"); got != "8655199155" {
			t.Fatalf("unexpected chat_id: %s", got)
		}
		if got := r.FormValue("caption"); got != "caption" {
			t.Fatalf("unexpected caption: %s", got)
		}
		file, _, err := r.FormFile("photo")
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		body, _ := io.ReadAll(file)
		if string(body) != "image-bytes" {
			t.Fatalf("unexpected file body: %q", string(body))
		}
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":456}}`))
	}))
	defer server.Close()
	adapter := newTestAdapter(t, server.URL)

	result, err := adapter.DispatchDeliveryStep(context.Background(), model.DeliveryDispatchStep{
		StepIndex: 1,
		Kind:      model.DeliveryDispatchStepImage,
		Channel:   "telegram",
		ChatID:    "8655199155",
		Message:   "caption",
		Image:     imagePath,
	})
	if err != nil {
		t.Fatalf("dispatch photo: %v", err)
	}
	if result.ProviderMessageID != "456" {
		t.Fatalf("unexpected provider message id: %+v", result)
	}
}

func TestTelegramAdapterClassifiesRouteError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"ok":false,"description":"Bad Request: chat not found"}`))
	}))
	defer server.Close()
	adapter := newTestAdapter(t, server.URL)

	_, err := adapter.DispatchDeliveryStep(context.Background(), model.DeliveryDispatchStep{
		StepIndex: 1,
		Kind:      model.DeliveryDispatchStepText,
		Channel:   "telegram",
		ChatID:    "missing",
		Message:   "hello",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	var kinded interface{ DeliveryErrorKind() string }
	if !errors.As(err, &kinded) {
		t.Fatalf("expected kinded error: %v", err)
	}
	if got := kinded.DeliveryErrorKind(); got != string(model.DeliveryErrorRoute) {
		t.Fatalf("expected route_error, got %s", got)
	}
}

func TestTelegramAdapterDoesNotClassifyUnauthorizedAsFallbackUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"ok":false,"description":"Unauthorized"}`))
	}))
	defer server.Close()
	adapter := newTestAdapter(t, server.URL)

	_, err := adapter.DispatchDeliveryStep(context.Background(), model.DeliveryDispatchStep{
		StepIndex: 1,
		Kind:      model.DeliveryDispatchStepText,
		Channel:   "telegram",
		ChatID:    "8655199155",
		Message:   "hello",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	var kinded interface{ DeliveryErrorKind() string }
	if !errors.As(err, &kinded) {
		t.Fatalf("expected kinded error: %v", err)
	}
	if got := kinded.DeliveryErrorKind(); got != string(model.DeliveryErrorPlatform) {
		t.Fatalf("expected platform_error, got %s", got)
	}
}

func TestTelegramAdapterSupportsConfiguredChannelAliases(t *testing.T) {
	adapter, err := telegramdelivery.NewAdapter(telegramdelivery.Config{
		Token:    "token",
		BaseURL:  "http://telegram.local",
		Channels: []string{"telegram", "telegram_work"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !adapter.SupportsDeliveryChannel("telegram_work") {
		t.Fatal("expected telegram_work to be supported")
	}
	if adapter.SupportsDeliveryChannel("qq") {
		t.Fatal("did not expect qq to be supported")
	}
}

func TestTelegramAdapterHealthUsesGetMeWithoutSendingMessage(t *testing.T) {
	var requestPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		if requestPath != "/bottoken/getMe" {
			t.Fatalf("unexpected path: %s", requestPath)
		}
		_, _ = w.Write([]byte(`{"ok":true,"result":{"id":7689386159,"is_bot":true,"first_name":"Dongri","username":"dongri0909bot"}}`))
	}))
	defer server.Close()
	adapter, err := telegramdelivery.NewAdapter(telegramdelivery.Config{
		Token:    "token",
		BaseURL:  server.URL,
		Channels: []string{"telegram", "dongri0909bot"},
	})
	if err != nil {
		t.Fatal(err)
	}

	items, err := adapter.CheckDeliveryAdapterHealth(context.Background(), query.DeliveryAdapterHealthFilter{TimeoutSeconds: 1})
	if err != nil {
		t.Fatalf("check health: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected two health items, got %#v", items)
	}
	for _, item := range items {
		if !item.Healthy || !item.Authenticated || item.AccountID != "7689386159" || item.AccountName != "dongri0909bot" {
			t.Fatalf("unexpected health item: %#v", item)
		}
		if item.SideEffect != "none" || item.Transport != "http" || !item.AccessTokenPresent {
			t.Fatalf("unexpected health metadata: %#v", item)
		}
	}
}

func newTestAdapter(t *testing.T, baseURL string) outport.DeliveryAdapter {
	t.Helper()
	adapter, err := telegramdelivery.NewAdapter(telegramdelivery.Config{
		Token:   "token",
		BaseURL: baseURL,
	})
	if err != nil {
		t.Fatal(err)
	}
	return adapter
}
