package query

type OutboundCutoverReadinessView struct {
	Ready                    bool                       `json:"ready"`
	Reason                   string                     `json:"reason"`
	OneBotReady              bool                       `json:"onebot_ready"`
	SmokeReady               bool                       `json:"smoke_ready"`
	ExecutionReady           bool                       `json:"execution_ready"`
	ExecutionOwner           string                     `json:"execution_owner"`
	LocalOutboxWorkerReady   bool                       `json:"local_outbox_worker_ready"`
	ExternalLeaseOutboxReady bool                       `json:"external_lease_outbox_ready"`
	QueueProvider            string                     `json:"queue_provider,omitempty"`
	QueueMode                string                     `json:"queue_mode,omitempty"`
	ExternalLeaseScope       string                     `json:"external_lease_scope,omitempty"`
	ExpectedOneBotChannels   []string                   `json:"expected_onebot_channels,omitempty"`
	MissingOneBotChannels    []string                   `json:"missing_onebot_channels,omitempty"`
	RuntimeConfigReadiness   RuntimeConfigReadinessView `json:"runtime_config_readiness"`
	DeliverySmokeReadiness   DeliverySmokeReadinessView `json:"delivery_smoke_readiness"`
	Blockers                 []string                   `json:"blockers,omitempty"`
	Attributes               map[string]string          `json:"attributes,omitempty"`
	Notes                    []string                   `json:"notes,omitempty"`
	SideEffect               string                     `json:"side_effect"`
}

type OutboundCutoverPlanView struct {
	Ready                     bool                         `json:"ready"`
	Decision                  string                       `json:"decision"`
	DesiredExecutionOwner     string                       `json:"desired_execution_owner"`
	RecommendedExecutionOwner string                       `json:"recommended_execution_owner"`
	CurrentExecutionOwner     string                       `json:"current_execution_owner"`
	Readiness                 OutboundCutoverReadinessView `json:"readiness"`
	RequiredChecks            []OutboundCutoverPlanStep    `json:"required_checks,omitempty"`
	EnableSteps               []OutboundCutoverPlanStep    `json:"enable_steps,omitempty"`
	VerificationSteps         []OutboundCutoverPlanStep    `json:"verification_steps,omitempty"`
	RollbackSteps             []OutboundCutoverPlanStep    `json:"rollback_steps,omitempty"`
	Blockers                  []string                     `json:"blockers,omitempty"`
	Attributes                map[string]string            `json:"attributes,omitempty"`
	Notes                     []string                     `json:"notes,omitempty"`
	SideEffect                string                       `json:"side_effect"`
}

type OutboundCutoverPlanStep struct {
	StepIndex int               `json:"step_index"`
	Phase     string            `json:"phase"`
	Action    string            `json:"action"`
	Detail    string            `json:"detail,omitempty"`
	Method    string            `json:"method,omitempty"`
	Endpoint  string            `json:"endpoint,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
}
