package command

type CheckOutboundCutoverReadinessCommand struct {
	Smoke CheckDeliverySmokeReadinessCommand
}

type PlanOutboundCutoverCommand struct {
	Readiness             CheckOutboundCutoverReadinessCommand
	DesiredExecutionOwner string
}
