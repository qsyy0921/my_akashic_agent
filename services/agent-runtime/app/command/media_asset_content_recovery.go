package command

type RecoverMediaAssetContentCommand struct {
	AssetID    string
	TargetID   string
	OperatorID string
	ApprovalID string
	MutationID string
	DryRun     bool
}
