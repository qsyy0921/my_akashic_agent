package onebotdelivery

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
)

func (a *Adapter) CheckDeliveryAdapterHealth(
	ctx context.Context,
	filter query.DeliveryAdapterHealthFilter,
) ([]query.DeliveryAdapterHealthView, error) {
	if a == nil {
		return nil, errors.New("onebot delivery adapter is nil")
	}
	items := make([]query.DeliveryAdapterHealthView, 0, len(a.endpoints))
	for channel, endpoint := range a.endpoints {
		items = append(items, a.checkEndpointHealth(ctx, channel, endpoint, filter))
	}
	return items, nil
}

func (a *Adapter) checkEndpointHealth(
	ctx context.Context,
	channel string,
	endpoint EndpointConfig,
	filter query.DeliveryAdapterHealthFilter,
) query.DeliveryAdapterHealthView {
	start := time.Now()
	checkedAt := start.UTC().Format(time.RFC3339Nano)
	probeCtx, cancel := context.WithTimeout(ctx, adapterHealthTimeout(filter))
	defer cancel()

	response, err := a.callLoginInfo(probeCtx, endpoint)
	latencyMs := int(time.Since(start).Milliseconds())
	item := query.DeliveryAdapterHealthView{
		Provider:           "onebot",
		Channel:            channel,
		Transport:          healthTransport(endpoint),
		Endpoint:           healthEndpoint(endpoint),
		CheckedAt:          checkedAt,
		LatencyMs:          latencyMs,
		SideEffect:         "none",
		AccessTokenPresent: strings.TrimSpace(endpoint.AccessToken) != "",
		Notes:              []string{"uses OneBot get_login_info; no messages are sent"},
	}
	if err != nil {
		item.ErrorKind = healthErrorKind(err)
		item.ErrorMessage = err.Error()
		item.Reachable = item.ErrorKind != "platform_timeout" && item.ErrorKind != "sender_unavailable"
		return item
	}
	item.Healthy = true
	item.Reachable = true
	item.Authenticated = true
	item.AccountID = response.Data.UserIDString()
	item.AccountName = strings.TrimSpace(response.Data.Nickname)
	return item
}

func (a *Adapter) callLoginInfo(ctx context.Context, endpoint EndpointConfig) (onebotResponse, error) {
	if strings.TrimSpace(endpoint.WebSocketURL) != "" {
		return a.callWebSocketResponse(ctx, endpoint, "get_login_info", map[string]any{})
	}
	return a.callHTTPResponse(ctx, endpoint, "get_login_info", map[string]any{})
}

func adapterHealthTimeout(filter query.DeliveryAdapterHealthFilter) time.Duration {
	seconds := filter.TimeoutSeconds
	if seconds <= 0 {
		seconds = 3
	}
	if seconds > 30 {
		seconds = 30
	}
	return time.Duration(seconds) * time.Second
}

func healthTransport(endpoint EndpointConfig) string {
	hasHTTP := strings.TrimSpace(endpoint.BaseURL) != ""
	hasWS := strings.TrimSpace(endpoint.WebSocketURL) != ""
	switch {
	case hasHTTP && hasWS:
		return "http+websocket"
	case hasWS:
		return "websocket"
	case hasHTTP:
		return "http"
	default:
		return "unconfigured"
	}
}

func healthEndpoint(endpoint EndpointConfig) string {
	if webSocketURL := strings.TrimSpace(endpoint.WebSocketURL); webSocketURL != "" {
		return redactHealthEndpoint(webSocketURL, endpoint.AccessToken)
	}
	return redactHealthEndpoint(strings.TrimSpace(endpoint.BaseURL), endpoint.AccessToken)
}

func redactHealthEndpoint(raw string, token string) string {
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return strings.ReplaceAll(raw, token, "<redacted>")
	}
	if parsed.User != nil {
		parsed.User = url.User("<redacted>")
	}
	queryValues := parsed.Query()
	if queryValues.Get("access_token") != "" {
		queryValues.Set("access_token", "<redacted>")
		parsed.RawQuery = queryValues.Encode()
	}
	result := parsed.String()
	if strings.TrimSpace(token) != "" {
		result = strings.ReplaceAll(result, token, "<redacted>")
	}
	return result
}

func healthErrorKind(err error) string {
	var kinded interface{ DeliveryErrorKind() string }
	if errors.As(err, &kinded) {
		return kinded.DeliveryErrorKind()
	}
	return "platform_error"
}
