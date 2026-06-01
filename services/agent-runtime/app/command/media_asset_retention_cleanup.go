package command

type CleanupMediaAssetRetentionCommand struct {
	AssetID               string
	Limit                 int
	ChannelKind           string
	AccountID             string
	ConversationID        string
	ConversationType      string
	SourceMessageID       string
	SourceMessageIDSuffix string
	Kind                  string
	Timestamp             string
	DefaultTTLHours       int
	EphemeralTTLHours     int
	TargetID              string
	OperatorID            string
	ApprovalID            string
	MutationID            string
	DryRun                bool
}
