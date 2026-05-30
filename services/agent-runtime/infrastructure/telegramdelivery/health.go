package telegramdelivery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/query"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

func (a *Adapter) CheckDeliveryAdapterHealth(
	ctx context.Context,
	filter query.DeliveryAdapterHealthFilter,
) ([]query.DeliveryAdapterHealthView, error) {
	if a == nil {
		return nil, errors.New("telegram delivery adapter is nil")
	}
	start := time.Now()
	checkedAt := start.UTC().Format(time.RFC3339Nano)
	probeCtx, cancel := context.WithTimeout(ctx, adapterHealthTimeout(filter))
	defer cancel()

	account, err := a.getMe(probeCtx)
	latencyMs := int(time.Since(start).Milliseconds())
	items := make([]query.DeliveryAdapterHealthView, 0, len(a.channels))
	for channel := range a.channels {
		item := query.DeliveryAdapterHealthView{
			Provider:           "telegram",
			Channel:            channel,
			Transport:          "http",
			Endpoint:           a.baseURL,
			CheckedAt:          checkedAt,
			LatencyMs:          latencyMs,
			SideEffect:         "none",
			AccessTokenPresent: a.token != "",
			Notes:              []string{"uses Telegram getMe; no messages are sent"},
		}
		if err != nil {
			item.ErrorKind = healthErrorKind(err)
			item.ErrorMessage = err.Error()
			item.Reachable = item.ErrorKind != "platform_timeout" && item.ErrorKind != "sender_unavailable"
		} else {
			item.Healthy = true
			item.Reachable = true
			item.Authenticated = true
			item.AccountID = fmt.Sprint(account.ID)
			item.AccountName = telegramAccountName(account)
			item.Attributes = map[string]string{"is_bot": fmt.Sprint(account.IsBot)}
		}
		items = append(items, item)
	}
	return items, nil
}

func (a *Adapter) getMe(ctx context.Context) (telegramAccount, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, a.methodURL("getMe"), nil)
	if err != nil {
		return telegramAccount{}, err
	}
	response, err := a.client.Do(request)
	if err != nil {
		return telegramAccount{}, classifyTransportError(err)
	}
	defer response.Body.Close()
	raw, readErr := io.ReadAll(io.LimitReader(response.Body, 64*1024))
	if readErr != nil {
		return telegramAccount{}, readErr
	}
	var payload telegramHealthResponse
	if err := json.Unmarshal(raw, &payload); err != nil {
		if response.StatusCode >= 200 && response.StatusCode < 300 {
			return telegramAccount{}, DeliveryError{Kind: model.DeliveryErrorPlatform, Message: "telegram response is not valid json"}
		}
		return telegramAccount{}, DeliveryError{
			Kind:    classifyTelegramFailure(response.StatusCode, string(raw)),
			Message: strings.TrimSpace(string(raw)),
		}
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 || !payload.OK {
		message := strings.TrimSpace(payload.Description)
		if message == "" {
			message = strings.TrimSpace(string(raw))
		}
		return telegramAccount{}, DeliveryError{
			Kind:    classifyTelegramFailure(response.StatusCode, message),
			Message: message,
		}
	}
	return payload.Result, nil
}

type telegramHealthResponse struct {
	OK          bool            `json:"ok"`
	Description string          `json:"description"`
	Result      telegramAccount `json:"result"`
}

type telegramAccount struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	Username  string `json:"username"`
}

func telegramAccountName(account telegramAccount) string {
	if username := strings.TrimSpace(account.Username); username != "" {
		return username
	}
	return strings.TrimSpace(account.FirstName)
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

func healthErrorKind(err error) string {
	var kinded interface{ DeliveryErrorKind() string }
	if errors.As(err, &kinded) {
		return kinded.DeliveryErrorKind()
	}
	return "platform_error"
}
