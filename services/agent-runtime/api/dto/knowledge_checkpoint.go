package dto

type UpsertKnowledgeCheckpointRequest struct {
	Cursor    int               `json:"cursor"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Timestamp string            `json:"timestamp,omitempty"`
}
