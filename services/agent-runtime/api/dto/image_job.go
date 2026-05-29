package dto

type CreateImageJobRequest struct {
	RequestID   string            `json:"request_id,omitempty"`
	Requester   ChannelDTO        `json:"requester"`
	RequesterID string            `json:"requester_id,omitempty"`
	Prompt      string            `json:"prompt"`
	Provider    string            `json:"provider,omitempty"`
	Model       string            `json:"model,omitempty"`
	Size        string            `json:"size,omitempty"`
	Count       int               `json:"count,omitempty"`
	MaxAttempts int               `json:"max_attempts,omitempty"`
	Timestamp   string            `json:"timestamp,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type ImageJobStateRequest struct {
	Results      []AttachmentDTO   `json:"results,omitempty"`
	ErrorMessage string            `json:"error_message,omitempty"`
	Timestamp    string            `json:"timestamp,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type ImageJobResponse struct {
	JobID        string            `json:"job_id"`
	RequestID    string            `json:"request_id,omitempty"`
	Requester    ChannelDTO        `json:"requester"`
	RequesterID  string            `json:"requester_id,omitempty"`
	Prompt       string            `json:"prompt"`
	Provider     string            `json:"provider,omitempty"`
	Model        string            `json:"model,omitempty"`
	Size         string            `json:"size,omitempty"`
	Count        int               `json:"count"`
	Status       string            `json:"status"`
	Attempts     int               `json:"attempts"`
	MaxAttempts  int               `json:"max_attempts"`
	Results      []AttachmentDTO   `json:"results,omitempty"`
	ErrorMessage string            `json:"error_message,omitempty"`
	CreatedAt    string            `json:"created_at"`
	UpdatedAt    string            `json:"updated_at"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

