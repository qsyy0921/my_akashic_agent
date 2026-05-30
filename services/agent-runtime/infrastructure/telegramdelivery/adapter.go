package telegramdelivery

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type Config struct {
	Token    string
	BaseURL  string
	Channels []string
	Client   *http.Client
}

type Adapter struct {
	token    string
	baseURL  string
	channels map[string]struct{}
	client   *http.Client
}

type DeliveryError struct {
	Kind    model.DeliveryErrorKind
	Message string
}

func (e DeliveryError) Error() string {
	return e.Message
}

func (e DeliveryError) DeliveryErrorKind() string {
	return string(model.NormalizeDeliveryErrorKind(string(e.Kind)))
}

func NewAdapter(config Config) (*Adapter, error) {
	token := strings.TrimSpace(config.Token)
	if token == "" {
		return nil, errors.New("telegram delivery adapter requires token")
	}
	baseURL := strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.telegram.org"
	}
	channels := config.Channels
	if len(channels) == 0 {
		channels = []string{"telegram"}
	}
	channelSet := make(map[string]struct{}, len(channels))
	for _, channel := range channels {
		channel = normalizeChannel(channel)
		if channel != "" {
			channelSet[channel] = struct{}{}
		}
	}
	if len(channelSet) == 0 {
		channelSet["telegram"] = struct{}{}
	}
	client := config.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &Adapter{
		token:    token,
		baseURL:  baseURL,
		channels: channelSet,
		client:   client,
	}, nil
}

func (a *Adapter) SupportsDeliveryChannel(channel string) bool {
	if a == nil {
		return false
	}
	_, ok := a.channels[normalizeChannel(channel)]
	return ok
}

func (a *Adapter) DispatchDeliveryStep(ctx context.Context, step model.DeliveryDispatchStep) (model.DeliveryDispatchResult, error) {
	if a == nil {
		return model.DeliveryDispatchResult{}, DeliveryError{
			Kind:    model.DeliveryErrorSenderUnavailable,
			Message: "telegram delivery adapter is nil",
		}
	}
	chatID := strings.TrimSpace(step.ChatID)
	if chatID == "" {
		return model.DeliveryDispatchResult{}, DeliveryError{
			Kind:    model.DeliveryErrorRoute,
			Message: "telegram delivery step missing chat_id",
		}
	}

	var messageID string
	var err error
	switch step.Kind {
	case model.DeliveryDispatchStepText:
		if strings.TrimSpace(step.Message) == "" {
			return model.DeliveryDispatchResult{}, DeliveryError{
				Kind:    model.DeliveryErrorValidation,
				Message: "telegram text step requires message",
			}
		}
		messageID, err = a.callJSON(ctx, "sendMessage", map[string]string{
			"chat_id": chatID,
			"text":    step.Message,
		})
	case model.DeliveryDispatchStepImage:
		image := strings.TrimSpace(step.Image)
		if image == "" {
			return model.DeliveryDispatchResult{}, DeliveryError{
				Kind:    model.DeliveryErrorValidation,
				Message: "telegram image step requires image",
			}
		}
		messageID, err = a.callMultipart(ctx, "sendPhoto", "photo", image, captionFields(chatID, step.Message))
	case model.DeliveryDispatchStepFile:
		file := strings.TrimSpace(step.File)
		if file == "" {
			return model.DeliveryDispatchResult{}, DeliveryError{
				Kind:    model.DeliveryErrorValidation,
				Message: "telegram file step requires file",
			}
		}
		messageID, err = a.callMultipart(ctx, "sendDocument", "document", file, captionFields(chatID, step.Message))
	default:
		return model.DeliveryDispatchResult{}, DeliveryError{
			Kind:    model.DeliveryErrorUnsupportedMedia,
			Message: fmt.Sprintf("telegram delivery step kind %q is unsupported", step.Kind),
		}
	}
	if err != nil {
		return model.DeliveryDispatchResult{}, err
	}
	return model.DeliveryDispatchResult{
		StepIndex:         step.StepIndex,
		Kind:              step.Kind,
		Channel:           step.Channel,
		ChatID:            step.ChatID,
		Status:            model.DeliveryDispatchSent,
		Provider:          "telegram",
		ProviderMessageID: messageID,
	}, nil
}

func captionFields(chatID string, caption string) map[string]string {
	fields := map[string]string{"chat_id": chatID}
	if caption != "" {
		fields["caption"] = caption
	}
	return fields
}

func (a *Adapter) callJSON(ctx context.Context, method string, payload map[string]string) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, a.methodURL(method), bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	return a.do(request)
}

func (a *Adapter) callMultipart(ctx context.Context, method string, mediaField string, mediaValue string, fields map[string]string) (string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return "", err
		}
	}
	if local, ok := resolveLocalFile(mediaValue); ok {
		file, err := os.Open(local)
		if err != nil {
			return "", DeliveryError{
				Kind:    model.DeliveryErrorUnsupportedMedia,
				Message: fmt.Sprintf("telegram media file unavailable: %s", mediaValue),
			}
		}
		defer file.Close()
		part, err := writer.CreateFormFile(mediaField, filepath.Base(local))
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(part, file); err != nil {
			return "", err
		}
	} else if err := writer.WriteField(mediaField, mediaValue); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, a.methodURL(method), &body)
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return a.do(request)
}

func (a *Adapter) do(request *http.Request) (string, error) {
	response, err := a.client.Do(request)
	if err != nil {
		return "", classifyTransportError(err)
	}
	defer response.Body.Close()
	raw, readErr := io.ReadAll(io.LimitReader(response.Body, 64*1024))
	if readErr != nil {
		return "", readErr
	}
	var payload telegramResponse
	if err := json.Unmarshal(raw, &payload); err != nil {
		if response.StatusCode >= 200 && response.StatusCode < 300 {
			return "", DeliveryError{
				Kind:    model.DeliveryErrorPlatform,
				Message: "telegram response is not valid json",
			}
		}
		return "", DeliveryError{
			Kind:    classifyTelegramFailure(response.StatusCode, string(raw)),
			Message: strings.TrimSpace(string(raw)),
		}
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 || !payload.OK {
		message := strings.TrimSpace(payload.Description)
		if message == "" {
			message = strings.TrimSpace(string(raw))
		}
		if payload.Parameters.RetryAfter > 0 {
			message = fmt.Sprintf("%s retry_after=%d", message, payload.Parameters.RetryAfter)
		}
		return "", DeliveryError{
			Kind:    classifyTelegramFailure(response.StatusCode, message),
			Message: message,
		}
	}
	if payload.Result.MessageID == 0 {
		return "", nil
	}
	return strconv.Itoa(payload.Result.MessageID), nil
}

func (a *Adapter) methodURL(method string) string {
	return fmt.Sprintf("%s/bot%s/%s", a.baseURL, a.token, method)
}

type telegramResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
	Result      struct {
		MessageID int `json:"message_id"`
	} `json:"result"`
	Parameters struct {
		RetryAfter int `json:"retry_after"`
	} `json:"parameters"`
}

func classifyTransportError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return DeliveryError{Kind: model.DeliveryErrorPlatformTimeout, Message: err.Error()}
	}
	if timeout, ok := err.(interface{ Timeout() bool }); ok && timeout.Timeout() {
		return DeliveryError{Kind: model.DeliveryErrorPlatformTimeout, Message: err.Error()}
	}
	return DeliveryError{Kind: model.DeliveryErrorPlatform, Message: err.Error()}
}

func classifyTelegramFailure(statusCode int, message string) model.DeliveryErrorKind {
	lower := strings.ToLower(message)
	switch {
	case statusCode == http.StatusTooManyRequests || strings.Contains(lower, "retry_after"):
		return model.DeliveryErrorPlatformTimeout
	case strings.Contains(lower, "chat not found"),
		strings.Contains(lower, "bot was blocked"),
		strings.Contains(lower, "user is deactivated"),
		strings.Contains(lower, "bot was kicked"),
		strings.Contains(lower, "have no rights"):
		return model.DeliveryErrorRoute
	case strings.Contains(lower, "wrong file identifier"),
		strings.Contains(lower, "wrong remote file identifier"),
		strings.Contains(lower, "file must be"):
		return model.DeliveryErrorUnsupportedMedia
	default:
		return model.DeliveryErrorPlatform
	}
}

func resolveLocalFile(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return "", false
	}
	if stat, err := os.Stat(value); err == nil && !stat.IsDir() {
		return value, true
	}
	return "", false
}

func normalizeChannel(channel string) string {
	return strings.ToLower(strings.TrimSpace(channel))
}
