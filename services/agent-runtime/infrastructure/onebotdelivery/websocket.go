package onebotdelivery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

const webSocketResponseReadLimit = 2 << 20

func (a *Adapter) callWebSocket(ctx context.Context, endpoint EndpointConfig, method string, payload map[string]any) (string, error) {
	webSocketURL := strings.TrimSpace(endpoint.WebSocketURL)
	if webSocketURL == "" {
		return "", DeliveryError{
			Kind:    model.DeliveryErrorSenderUnavailable,
			Message: "onebot websocket endpoint is empty",
		}
	}
	ctx, cancel := withDefaultTimeout(ctx, 30*time.Second)
	defer cancel()

	echo := onebotEcho()
	request := onebotActionRequest{
		Action: method,
		Params: payload,
		Echo:   echo,
	}
	header := http.Header{}
	if token := strings.TrimSpace(endpoint.AccessToken); token != "" {
		header.Set("Authorization", "Bearer "+token)
		webSocketURL = webSocketURLWithAccessToken(webSocketURL, token)
	}

	conn, response, err := a.webSocketDialer.DialContext(ctx, webSocketURL, header)
	if err != nil {
		return "", classifyWebSocketDialError(err, response)
	}
	defer conn.Close()
	conn.SetReadLimit(webSocketResponseReadLimit)
	applyWebSocketDeadline(ctx, conn)

	if err := conn.WriteJSON(request); err != nil {
		return "", classifyTransportError(err)
	}

	for {
		var response onebotResponse
		if err := conn.ReadJSON(&response); err != nil {
			return "", classifyTransportError(err)
		}
		if !echoMatches(response.Echo, echo) {
			continue
		}
		if response.Retcode != 0 {
			message := strings.TrimSpace(response.Message)
			if message == "" {
				message = strings.TrimSpace(response.Wording)
			}
			if message == "" {
				raw, _ := json.Marshal(response)
				message = strings.TrimSpace(string(raw))
			}
			return "", DeliveryError{
				Kind:    classifyOneBotFailure(http.StatusOK, message),
				Message: message,
			}
		}
		return response.Data.MessageIDString(), nil
	}
}

type onebotActionRequest struct {
	Action string         `json:"action"`
	Params map[string]any `json:"params,omitempty"`
	Echo   string         `json:"echo"`
}

func withDefaultTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if ctx == nil {
		return context.WithTimeout(context.Background(), timeout)
	}
	if _, ok := ctx.Deadline(); ok {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, timeout)
}

func applyWebSocketDeadline(ctx context.Context, conn *websocket.Conn) {
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetReadDeadline(deadline)
		_ = conn.SetWriteDeadline(deadline)
	}
}

func webSocketURLWithAccessToken(raw string, token string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	query := parsed.Query()
	if query.Get("access_token") == "" {
		query.Set("access_token", token)
		parsed.RawQuery = query.Encode()
	}
	return parsed.String()
}

func onebotEcho() string {
	return "akashic-" + strconv.FormatInt(time.Now().UnixNano(), 10)
}

func echoMatches(value any, expected string) bool {
	switch typed := value.(type) {
	case string:
		return typed == expected
	case float64:
		return strconv.FormatInt(int64(typed), 10) == expected
	case int:
		return strconv.Itoa(typed) == expected
	default:
		return fmt.Sprint(value) == expected
	}
}

func classifyWebSocketDialError(err error, response *http.Response) error {
	if response == nil {
		return classifyTransportError(err)
	}
	switch response.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return DeliveryError{Kind: model.DeliveryErrorPlatform, Message: err.Error()}
	case http.StatusTooManyRequests, http.StatusGatewayTimeout:
		return DeliveryError{Kind: model.DeliveryErrorPlatformTimeout, Message: err.Error()}
	default:
		return classifyTransportError(err)
	}
}
