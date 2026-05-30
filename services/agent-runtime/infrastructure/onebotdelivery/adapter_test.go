package onebotdelivery_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/onebotdelivery"
)

func TestOneBotAdapterSendsPrivateTextMessage(t *testing.T) {
	var requestBody map[string]any
	var authHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/send_private_msg" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		authHeader = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"status":"ok","retcode":0,"data":{"message_id":123}}`))
	}))
	defer server.Close()
	adapter := newTestAdapter(t, server.URL)

	result, err := adapter.DispatchDeliveryStep(context.Background(), model.DeliveryDispatchStep{
		StepIndex:        1,
		Kind:             model.DeliveryDispatchStepText,
		Channel:          "qq_2365524513",
		ChatID:           "1049511700",
		ConversationType: model.ConversationTypePrivate,
		Message:          "hello",
	})
	if err != nil {
		t.Fatalf("dispatch text: %v", err)
	}
	if requestBody["user_id"] != float64(1049511700) || requestBody["message"] != "hello" {
		t.Fatalf("unexpected request body: %+v", requestBody)
	}
	if authHeader != "Bearer token" {
		t.Fatalf("unexpected authorization header: %q", authHeader)
	}
	if result.Provider != "onebot" || result.ProviderMessageID != "123" || result.Status != model.DeliveryDispatchSent {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestOneBotAdapterSendsGroupImageAsMessageSegment(t *testing.T) {
	dir := t.TempDir()
	imagePath := filepath.Join(dir, "image.png")
	if err := os.WriteFile(imagePath, []byte("image-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	var requestBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/send_group_msg" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"status":"ok","retcode":0,"data":{"message_id":"msg-2"}}`))
	}))
	defer server.Close()
	adapter := newTestAdapter(t, server.URL)

	result, err := adapter.DispatchDeliveryStep(context.Background(), model.DeliveryDispatchStep{
		StepIndex:        1,
		Kind:             model.DeliveryDispatchStepImage,
		Channel:          "qq_2365524513",
		ChatID:           "27234224",
		ConversationType: model.ConversationTypeGroup,
		Message:          "caption",
		Image:            imagePath,
	})
	if err != nil {
		t.Fatalf("dispatch image: %v", err)
	}
	if requestBody["group_id"] != float64(27234224) {
		t.Fatalf("unexpected group id: %+v", requestBody)
	}
	segments, ok := requestBody["message"].([]any)
	if !ok || len(segments) != 2 {
		t.Fatalf("unexpected message segments: %#v", requestBody["message"])
	}
	imageSegment := segments[1].(map[string]any)
	if imageSegment["type"] != "image" {
		t.Fatalf("unexpected image segment: %#v", imageSegment)
	}
	data := imageSegment["data"].(map[string]any)
	expected := "base64://" + base64.StdEncoding.EncodeToString([]byte("image-bytes"))
	if data["file"] != expected {
		t.Fatalf("unexpected image file payload")
	}
	if result.ProviderMessageID != "msg-2" {
		t.Fatalf("unexpected provider message id: %+v", result)
	}
}

func TestOneBotAdapterUploadsPrivateFileAfterOptionalText(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(filePath, []byte("file-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	paths := make([]string, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		switch r.URL.Path {
		case "/send_private_msg":
			if body["message"] != "caption" {
				t.Fatalf("unexpected text body: %+v", body)
			}
			_, _ = w.Write([]byte(`{"status":"ok","retcode":0,"data":{"message_id":11}}`))
		case "/upload_private_file":
			if body["name"] != "note.txt" || !strings.HasPrefix(body["file"].(string), "base64://") {
				t.Fatalf("unexpected upload body: %+v", body)
			}
			_, _ = w.Write([]byte(`{"status":"ok","retcode":0,"data":{}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	adapter := newTestAdapter(t, server.URL)

	result, err := adapter.DispatchDeliveryStep(context.Background(), model.DeliveryDispatchStep{
		StepIndex:        1,
		Kind:             model.DeliveryDispatchStepFile,
		Channel:          "qq_2365524513",
		ChatID:           "1049511700",
		ConversationType: model.ConversationTypePrivate,
		Message:          "caption",
		File:             filePath,
	})
	if err != nil {
		t.Fatalf("dispatch file: %v", err)
	}
	if strings.Join(paths, ",") != "/send_private_msg,/upload_private_file" {
		t.Fatalf("unexpected request order: %v", paths)
	}
	if result.Attributes["text_message_id"] != "11" {
		t.Fatalf("expected text message id attribute: %+v", result)
	}
}

func TestOneBotAdapterInfersGroupFromGQQPrefix(t *testing.T) {
	var requestBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/send_group_msg" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"status":"ok","retcode":0,"data":{"message_id":1}}`))
	}))
	defer server.Close()
	adapter := newTestAdapter(t, server.URL)

	_, err := adapter.DispatchDeliveryStep(context.Background(), model.DeliveryDispatchStep{
		StepIndex: 1,
		Kind:      model.DeliveryDispatchStepText,
		Channel:   "qq_2365524513",
		ChatID:    "gqq:27234224",
		Message:   "hello",
	})
	if err != nil {
		t.Fatalf("dispatch gqq group: %v", err)
	}
	if requestBody["group_id"] != float64(27234224) {
		t.Fatalf("unexpected request body: %+v", requestBody)
	}
}

func TestOneBotAdapterClassifiesRouteError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"status":"failed","retcode":100,"message":"group not found"}`))
	}))
	defer server.Close()
	adapter := newTestAdapter(t, server.URL)

	_, err := adapter.DispatchDeliveryStep(context.Background(), model.DeliveryDispatchStep{
		StepIndex:        1,
		Kind:             model.DeliveryDispatchStepText,
		Channel:          "qq_2365524513",
		ChatID:           "missing",
		ConversationType: model.ConversationTypeGroup,
		Message:          "hello",
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

func TestOneBotAdapterSupportsConfiguredChannelAliases(t *testing.T) {
	adapter, err := onebotdelivery.NewAdapter(onebotdelivery.Config{
		Endpoints: map[string]onebotdelivery.EndpointConfig{
			"qq_1049511700": {BaseURL: "http://onebot-a.local"},
			"qq_2365524513": {BaseURL: "http://onebot-b.local"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !adapter.SupportsDeliveryChannel("qq_1049511700") {
		t.Fatal("expected qq_1049511700 to be supported")
	}
	if adapter.SupportsDeliveryChannel("telegram") {
		t.Fatal("did not expect telegram to be supported")
	}
}

func TestOneBotAdapterDispatchesViaWebSocketAction(t *testing.T) {
	upgrader := websocket.Upgrader{}
	var actionRequest map[string]any
	var authHeader string
	var accessToken string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		accessToken = r.URL.Query().Get("access_token")
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		if err := conn.ReadJSON(&actionRequest); err != nil {
			t.Fatal(err)
		}
		if err := conn.WriteJSON(map[string]any{
			"post_type": "message",
			"message":   "ignored event",
		}); err != nil {
			t.Fatal(err)
		}
		if err := conn.WriteJSON(map[string]any{
			"status":  "ok",
			"retcode": 0,
			"echo":    actionRequest["echo"],
			"data": map[string]any{
				"message_id": 42,
			},
		}); err != nil {
			t.Fatal(err)
		}
	}))
	defer server.Close()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	adapter, err := onebotdelivery.NewAdapter(onebotdelivery.Config{
		Endpoints: map[string]onebotdelivery.EndpointConfig{
			"qq_2365524513": {
				WebSocketURL: wsURL,
				AccessToken:  "token",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := adapter.DispatchDeliveryStep(context.Background(), model.DeliveryDispatchStep{
		StepIndex:        1,
		Kind:             model.DeliveryDispatchStepText,
		Channel:          "qq_2365524513",
		ChatID:           "1049511700",
		ConversationType: model.ConversationTypePrivate,
		Message:          "hello over ws",
	})
	if err != nil {
		t.Fatalf("dispatch websocket text: %v", err)
	}

	if authHeader != "Bearer token" || accessToken != "token" {
		t.Fatalf("expected token in websocket header and query, got header=%q query=%q", authHeader, accessToken)
	}
	if actionRequest["action"] != "send_private_msg" {
		t.Fatalf("unexpected action request: %+v", actionRequest)
	}
	params := actionRequest["params"].(map[string]any)
	if params["user_id"] != float64(1049511700) || params["message"] != "hello over ws" {
		t.Fatalf("unexpected action params: %+v", params)
	}
	if result.ProviderMessageID != "42" || result.Provider != "onebot" {
		t.Fatalf("unexpected dispatch result: %+v", result)
	}
}

func TestOneBotAdapterHealthUsesHTTPLoginInfoWithoutSendingMessage(t *testing.T) {
	var requestPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		if requestPath != "/get_login_info" {
			t.Fatalf("unexpected path: %s", requestPath)
		}
		_, _ = w.Write([]byte(`{"status":"ok","retcode":0,"data":{"user_id":2365524513,"nickname":"bot-236"}}`))
	}))
	defer server.Close()
	adapter, err := onebotdelivery.NewAdapter(onebotdelivery.Config{
		Endpoints: map[string]onebotdelivery.EndpointConfig{
			"qq_2365524513": {
				BaseURL:     server.URL,
				AccessToken: "token",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	items, err := adapter.CheckDeliveryAdapterHealth(context.Background(), query.DeliveryAdapterHealthFilter{TimeoutSeconds: 1})
	if err != nil {
		t.Fatalf("check health: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one health item, got %#v", items)
	}
	item := items[0]
	if !item.Healthy || !item.Authenticated || item.AccountID != "2365524513" || item.AccountName != "bot-236" {
		t.Fatalf("unexpected health item: %#v", item)
	}
	if item.SideEffect != "none" || item.Transport != "http" || !item.AccessTokenPresent {
		t.Fatalf("unexpected health metadata: %#v", item)
	}
}

func TestOneBotAdapterHealthUsesWebSocketLoginInfoWithoutSendingMessage(t *testing.T) {
	upgrader := websocket.Upgrader{}
	var actionRequest map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		if err := conn.ReadJSON(&actionRequest); err != nil {
			t.Fatal(err)
		}
		if err := conn.WriteJSON(map[string]any{
			"status":  "ok",
			"retcode": 0,
			"echo":    actionRequest["echo"],
			"data": map[string]any{
				"user_id":  1049511700,
				"nickname": "bot-104",
			},
		}); err != nil {
			t.Fatal(err)
		}
	}))
	defer server.Close()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	adapter, err := onebotdelivery.NewAdapter(onebotdelivery.Config{
		Endpoints: map[string]onebotdelivery.EndpointConfig{
			"qq_1049511700": {WebSocketURL: wsURL},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	items, err := adapter.CheckDeliveryAdapterHealth(context.Background(), query.DeliveryAdapterHealthFilter{TimeoutSeconds: 1})
	if err != nil {
		t.Fatalf("check websocket health: %v", err)
	}
	if actionRequest["action"] != "get_login_info" {
		t.Fatalf("unexpected websocket action: %#v", actionRequest)
	}
	if len(items) != 1 || !items[0].Healthy || items[0].AccountID != "1049511700" || items[0].Transport != "websocket" {
		t.Fatalf("unexpected health items: %#v", items)
	}
}

func newTestAdapter(t *testing.T, baseURL string) outport.DeliveryAdapter {
	t.Helper()
	adapter, err := onebotdelivery.NewAdapter(onebotdelivery.Config{
		Endpoints: map[string]onebotdelivery.EndpointConfig{
			"qq_2365524513": {
				BaseURL:     baseURL,
				AccessToken: "token",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return adapter
}
