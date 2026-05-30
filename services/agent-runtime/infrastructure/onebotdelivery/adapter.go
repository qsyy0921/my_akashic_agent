package onebotdelivery

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type EndpointConfig struct {
	BaseURL     string
	AccessToken string
}

type Config struct {
	Endpoints map[string]EndpointConfig
	Client    *http.Client
}

type Adapter struct {
	endpoints map[string]EndpointConfig
	client    *http.Client
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
	endpoints := make(map[string]EndpointConfig, len(config.Endpoints))
	for channel, endpoint := range config.Endpoints {
		channel = normalizeChannel(channel)
		baseURL := strings.TrimRight(strings.TrimSpace(endpoint.BaseURL), "/")
		if channel == "" || baseURL == "" {
			continue
		}
		endpoints[channel] = EndpointConfig{
			BaseURL:     baseURL,
			AccessToken: strings.TrimSpace(endpoint.AccessToken),
		}
	}
	if len(endpoints) == 0 {
		return nil, errors.New("onebot delivery adapter requires endpoints")
	}
	client := config.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &Adapter{endpoints: endpoints, client: client}, nil
}

func (a *Adapter) SupportsDeliveryChannel(channel string) bool {
	if a == nil {
		return false
	}
	_, ok := a.endpoints[normalizeChannel(channel)]
	return ok
}

func (a *Adapter) DispatchDeliveryStep(ctx context.Context, step model.DeliveryDispatchStep) (model.DeliveryDispatchResult, error) {
	if a == nil {
		return model.DeliveryDispatchResult{}, DeliveryError{
			Kind:    model.DeliveryErrorSenderUnavailable,
			Message: "onebot delivery adapter is nil",
		}
	}
	endpoint, ok := a.endpoints[normalizeChannel(step.Channel)]
	if !ok {
		return model.DeliveryDispatchResult{}, DeliveryError{
			Kind:    model.DeliveryErrorSenderUnavailable,
			Message: fmt.Sprintf("onebot delivery adapter unavailable for channel %q", step.Channel),
		}
	}
	target, err := resolveTarget(step)
	if err != nil {
		return model.DeliveryDispatchResult{}, err
	}

	var providerMessageID string
	attributes := map[string]string{}
	switch step.Kind {
	case model.DeliveryDispatchStepText:
		if strings.TrimSpace(step.Message) == "" {
			return model.DeliveryDispatchResult{}, DeliveryError{
				Kind:    model.DeliveryErrorValidation,
				Message: "onebot text step requires message",
			}
		}
		providerMessageID, err = a.sendMessage(ctx, endpoint, target, step.Message)
	case model.DeliveryDispatchStepImage:
		image := strings.TrimSpace(step.Image)
		if image == "" {
			return model.DeliveryDispatchResult{}, DeliveryError{
				Kind:    model.DeliveryErrorValidation,
				Message: "onebot image step requires image",
			}
		}
		providerMessageID, err = a.sendImage(ctx, endpoint, target, step.Message, image)
	case model.DeliveryDispatchStepFile:
		file := strings.TrimSpace(step.File)
		if file == "" {
			return model.DeliveryDispatchResult{}, DeliveryError{
				Kind:    model.DeliveryErrorValidation,
				Message: "onebot file step requires file",
			}
		}
		var textMessageID string
		if strings.TrimSpace(step.Message) != "" {
			textMessageID, err = a.sendMessage(ctx, endpoint, target, step.Message)
			if err != nil {
				return model.DeliveryDispatchResult{}, err
			}
			if textMessageID != "" {
				attributes["text_message_id"] = textMessageID
			}
		}
		providerMessageID, err = a.uploadFile(ctx, endpoint, target, file)
	default:
		return model.DeliveryDispatchResult{}, DeliveryError{
			Kind:    model.DeliveryErrorUnsupportedMedia,
			Message: fmt.Sprintf("onebot delivery step kind %q is unsupported", step.Kind),
		}
	}
	if err != nil {
		return model.DeliveryDispatchResult{}, err
	}
	if len(attributes) == 0 {
		attributes = nil
	}
	return model.DeliveryDispatchResult{
		StepIndex:         step.StepIndex,
		Kind:              step.Kind,
		Channel:           step.Channel,
		ChatID:            step.ChatID,
		Status:            model.DeliveryDispatchSent,
		Provider:          "onebot",
		ProviderMessageID: providerMessageID,
		Attributes:        attributes,
	}, nil
}

type targetRef struct {
	Kind string
	ID   string
}

func resolveTarget(step model.DeliveryDispatchStep) (targetRef, error) {
	chatID := strings.TrimSpace(step.ChatID)
	if chatID == "" {
		return targetRef{}, DeliveryError{
			Kind:    model.DeliveryErrorRoute,
			Message: "onebot delivery step missing chat_id",
		}
	}
	if strings.HasPrefix(chatID, "gqq:") {
		id := strings.TrimSpace(strings.TrimPrefix(chatID, "gqq:"))
		if id == "" {
			return targetRef{}, DeliveryError{Kind: model.DeliveryErrorRoute, Message: "onebot group chat_id is empty"}
		}
		return targetRef{Kind: "group", ID: id}, nil
	}
	switch step.ConversationType {
	case model.ConversationTypeGroup:
		return targetRef{Kind: "group", ID: chatID}, nil
	case model.ConversationTypePrivate, "":
		return targetRef{Kind: "private", ID: chatID}, nil
	default:
		return targetRef{}, DeliveryError{
			Kind:    model.DeliveryErrorRoute,
			Message: fmt.Sprintf("onebot unsupported conversation_type %q", step.ConversationType),
		}
	}
}

func (a *Adapter) sendMessage(ctx context.Context, endpoint EndpointConfig, target targetRef, text string) (string, error) {
	payload := targetPayload(target)
	payload["message"] = text
	return a.call(ctx, endpoint, sendMessageMethod(target), payload)
}

func (a *Adapter) sendImage(ctx context.Context, endpoint EndpointConfig, target targetRef, caption string, image string) (string, error) {
	file, err := onebotMediaFile(image)
	if err != nil {
		return "", err
	}
	message := make([]map[string]any, 0, 2)
	if strings.TrimSpace(caption) != "" {
		message = append(message, map[string]any{
			"type": "text",
			"data": map[string]string{"text": caption},
		})
	}
	message = append(message, map[string]any{
		"type": "image",
		"data": map[string]string{"file": file},
	})
	payload := targetPayload(target)
	payload["message"] = message
	return a.call(ctx, endpoint, sendMessageMethod(target), payload)
}

func (a *Adapter) uploadFile(ctx context.Context, endpoint EndpointConfig, target targetRef, filePath string) (string, error) {
	file, err := onebotMediaFile(filePath)
	if err != nil {
		return "", err
	}
	payload := targetPayload(target)
	payload["file"] = file
	payload["name"] = mediaName(filePath)
	return a.call(ctx, endpoint, uploadFileMethod(target), payload)
}

func sendMessageMethod(target targetRef) string {
	if target.Kind == "group" {
		return "send_group_msg"
	}
	return "send_private_msg"
}

func uploadFileMethod(target targetRef) string {
	if target.Kind == "group" {
		return "upload_group_file"
	}
	return "upload_private_file"
}

func targetPayload(target targetRef) map[string]any {
	if target.Kind == "group" {
		return map[string]any{"group_id": targetIDValue(target.ID)}
	}
	return map[string]any{"user_id": targetIDValue(target.ID)}
}

func targetIDValue(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	id, err := strconv.ParseInt(value, 10, 64)
	if err == nil {
		return id
	}
	return value
}

func (a *Adapter) call(ctx context.Context, endpoint EndpointConfig, method string, payload map[string]any) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.BaseURL+"/"+method, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	if endpoint.AccessToken != "" {
		request.Header.Set("Authorization", "Bearer "+endpoint.AccessToken)
	}
	response, err := a.client.Do(request)
	if err != nil {
		return "", classifyTransportError(err)
	}
	defer response.Body.Close()
	raw, readErr := io.ReadAll(io.LimitReader(response.Body, 64*1024))
	if readErr != nil {
		return "", readErr
	}
	var payloadResp onebotResponse
	if err := json.Unmarshal(raw, &payloadResp); err != nil {
		if response.StatusCode >= 200 && response.StatusCode < 300 {
			return "", DeliveryError{Kind: model.DeliveryErrorPlatform, Message: "onebot response is not valid json"}
		}
		return "", DeliveryError{
			Kind:    classifyOneBotFailure(response.StatusCode, string(raw)),
			Message: strings.TrimSpace(string(raw)),
		}
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 || payloadResp.Retcode != 0 {
		message := strings.TrimSpace(payloadResp.Message)
		if message == "" {
			message = strings.TrimSpace(payloadResp.Wording)
		}
		if message == "" {
			message = strings.TrimSpace(string(raw))
		}
		return "", DeliveryError{
			Kind:    classifyOneBotFailure(response.StatusCode, message),
			Message: message,
		}
	}
	return payloadResp.Data.MessageIDString(), nil
}

type onebotResponse struct {
	Status  string         `json:"status"`
	Retcode int            `json:"retcode"`
	Message string         `json:"message"`
	Wording string         `json:"wording"`
	Data    onebotRespData `json:"data"`
}

type onebotRespData struct {
	MessageID any `json:"message_id"`
}

func (d onebotRespData) MessageIDString() string {
	switch value := d.MessageID.(type) {
	case string:
		return value
	case float64:
		return strconv.FormatInt(int64(value), 10)
	case int:
		return strconv.Itoa(value)
	default:
		return ""
	}
}

func onebotMediaFile(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", DeliveryError{Kind: model.DeliveryErrorUnsupportedMedia, Message: "onebot media path is empty"}
	}
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") || strings.HasPrefix(value, "base64://") {
		return value, nil
	}
	stat, err := os.Stat(value)
	if err != nil || stat.IsDir() {
		return "", DeliveryError{
			Kind:    model.DeliveryErrorUnsupportedMedia,
			Message: fmt.Sprintf("onebot media file unavailable: %s", value),
		}
	}
	raw, err := os.ReadFile(value)
	if err != nil {
		return "", DeliveryError{
			Kind:    model.DeliveryErrorUnsupportedMedia,
			Message: fmt.Sprintf("onebot media file unavailable: %s", value),
		}
	}
	return "base64://" + base64.StdEncoding.EncodeToString(raw), nil
}

func mediaName(value string) string {
	if name := filepath.Base(strings.TrimSpace(value)); name != "." && name != "" {
		return name
	}
	extension := mime.TypeByExtension(filepath.Ext(value))
	if extension != "" {
		return "attachment" + filepath.Ext(value)
	}
	return "attachment"
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

func classifyOneBotFailure(statusCode int, message string) model.DeliveryErrorKind {
	lower := strings.ToLower(message)
	switch {
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return model.DeliveryErrorPlatform
	case statusCode == http.StatusTooManyRequests || strings.Contains(lower, "timeout"):
		return model.DeliveryErrorPlatformTimeout
	case strings.Contains(lower, "group") && strings.Contains(lower, "not"),
		strings.Contains(lower, "群") && strings.Contains(lower, "不存在"),
		strings.Contains(lower, "user") && strings.Contains(lower, "not"),
		strings.Contains(lower, "好友") && strings.Contains(lower, "不存在"):
		return model.DeliveryErrorRoute
	case strings.Contains(lower, "file"),
		strings.Contains(lower, "image"),
		strings.Contains(lower, "unsupported"),
		strings.Contains(lower, "不支持"):
		return model.DeliveryErrorUnsupportedMedia
	default:
		return model.DeliveryErrorPlatform
	}
}

func normalizeChannel(channel string) string {
	return strings.ToLower(strings.TrimSpace(channel))
}
