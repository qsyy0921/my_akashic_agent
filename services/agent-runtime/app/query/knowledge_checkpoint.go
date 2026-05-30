package query

type KnowledgeCheckpointView struct {
	CheckpointID string            `json:"checkpoint_id"`
	Cursor       int               `json:"cursor"`
	UpdatedAt    string            `json:"updated_at"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}
