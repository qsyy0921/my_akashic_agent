package main

import (
	"os"
	"strings"

	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/onebotdelivery"
)

func telegramBotTokenFromEnv() string {
	if token := strings.TrimSpace(os.Getenv("AKASHIC_TELEGRAM_BOT_TOKEN")); token != "" {
		return token
	}
	return strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
}

func deliveryAdapterDiagnosticsFromEnv() []query.DeliveryAdapterDiagnosticsView {
	items := make([]query.DeliveryAdapterDiagnosticsView, 0)
	if token := telegramBotTokenFromEnv(); token != "" {
		endpoint := strings.TrimSpace(os.Getenv("AKASHIC_TELEGRAM_API_BASE_URL"))
		notes := []string(nil)
		if endpoint == "" {
			endpoint = "https://api.telegram.org"
			notes = append(notes, "using default Telegram Bot API endpoint")
		}
		for _, channel := range csvEnvOrDefault("AKASHIC_TELEGRAM_CHANNELS", []string{"telegram"}) {
			channel = strings.TrimSpace(channel)
			if channel == "" {
				continue
			}
			items = append(items, query.DeliveryAdapterDiagnosticsView{
				Provider:              "telegram",
				Channel:               channel,
				Transport:             "http",
				Enabled:               true,
				EndpointConfigured:    true,
				AccessTokenConfigured: token != "",
				Endpoint:              redactQueueDSN(endpoint),
				Notes:                 append([]string(nil), notes...),
			})
		}
	}

	onebotEndpoints := onebotEndpointsFromEnv()
	for _, channel := range sortedMapKeys(onebotEndpoints) {
		endpoint := onebotEndpoints[channel]
		items = append(items, query.DeliveryAdapterDiagnosticsView{
			Provider:              "onebot",
			Channel:               channel,
			Transport:             onebotTransport(endpoint),
			Enabled:               true,
			EndpointConfigured:    strings.TrimSpace(endpoint.BaseURL) != "" || strings.TrimSpace(endpoint.WebSocketURL) != "",
			AccessTokenConfigured: strings.TrimSpace(endpoint.AccessToken) != "",
			Endpoint:              onebotDiagnosticEndpoint(endpoint),
			Notes:                 onebotDiagnosticNotes(endpoint),
		})
	}
	return items
}

func deliveryAdapterHealthProbes(adapters []outport.DeliveryAdapter) []outport.DeliveryAdapterHealthProbe {
	probes := make([]outport.DeliveryAdapterHealthProbe, 0, len(adapters))
	for _, adapter := range adapters {
		probe, ok := adapter.(outport.DeliveryAdapterHealthProbe)
		if ok {
			probes = append(probes, probe)
		}
	}
	return probes
}

func onebotTransport(endpoint onebotdelivery.EndpointConfig) string {
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

func onebotDiagnosticEndpoint(endpoint onebotdelivery.EndpointConfig) string {
	if webSocketURL := strings.TrimSpace(endpoint.WebSocketURL); webSocketURL != "" {
		return redactQueueDSN(webSocketURL)
	}
	return redactQueueDSN(strings.TrimSpace(endpoint.BaseURL))
}

func onebotDiagnosticNotes(endpoint onebotdelivery.EndpointConfig) []string {
	if strings.TrimSpace(endpoint.BaseURL) != "" && strings.TrimSpace(endpoint.WebSocketURL) != "" {
		return []string{"websocket transport is preferred when both HTTP and WebSocket endpoints are configured"}
	}
	return nil
}
