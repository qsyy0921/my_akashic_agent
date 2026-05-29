package model

type LoopAction string

const (
	LoopActionAllow       LoopAction = "allow"
	LoopActionObserveOnly LoopAction = "observe_only"
	LoopActionDrop        LoopAction = "drop"
)

type LoopDecision struct {
	Action LoopAction
	Reason string
}

