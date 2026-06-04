package query

type AgentJobExternalLeaseReadinessView struct {
	Ready                   bool                         `json:"ready"`
	Reason                  string                       `json:"reason"`
	ExternalLeaseReady      bool                         `json:"external_lease_ready"`
	AgentJobResultAckReady  bool                         `json:"agent_job_result_ack_ready"`
	StrictLeaseTokenEnabled bool                         `json:"strict_lease_token_enabled"`
	AgentJobWorkerReady     bool                         `json:"agent_job_worker_ready"`
	ExecutionOwner          string                       `json:"execution_owner,omitempty"`
	QueueProvider           string                       `json:"queue_provider,omitempty"`
	QueueMode               string                       `json:"queue_mode,omitempty"`
	ExecutionScope          string                       `json:"execution_scope,omitempty"`
	AllowedWorkKinds        []string                     `json:"allowed_work_kinds,omitempty"`
	BlockedWorkKinds        []QueueExternalLeaseBlock    `json:"blocked_work_kinds,omitempty"`
	RequiredChecks          []QueueExternalLeaseCheck    `json:"required_checks,omitempty"`
	WorkerCoverage          []AgentJobWorkerCoverageView `json:"worker_coverage,omitempty"`
	Blockers                []string                     `json:"blockers,omitempty"`
	Attributes              map[string]string            `json:"attributes,omitempty"`
	Notes                   []string                     `json:"notes,omitempty"`
	SideEffect              string                       `json:"side_effect"`
}

type AgentJobExternalLeasePlanView struct {
	Ready                     bool                               `json:"ready"`
	Decision                  string                             `json:"decision"`
	DesiredExecutionOwner     string                             `json:"desired_execution_owner"`
	RecommendedExecutionOwner string                             `json:"recommended_execution_owner"`
	CurrentExecutionOwner     string                             `json:"current_execution_owner"`
	Readiness                 AgentJobExternalLeaseReadinessView `json:"readiness"`
	RequiredChecks            []AgentJobExternalLeasePlanStep    `json:"required_checks,omitempty"`
	EnableSteps               []AgentJobExternalLeasePlanStep    `json:"enable_steps,omitempty"`
	VerificationSteps         []AgentJobExternalLeasePlanStep    `json:"verification_steps,omitempty"`
	RollbackSteps             []AgentJobExternalLeasePlanStep    `json:"rollback_steps,omitempty"`
	Blockers                  []string                           `json:"blockers,omitempty"`
	Attributes                map[string]string                  `json:"attributes,omitempty"`
	Notes                     []string                           `json:"notes,omitempty"`
	SideEffect                string                             `json:"side_effect"`
}

type AgentJobExternalLeasePlanStep struct {
	StepIndex int               `json:"step_index"`
	Phase     string            `json:"phase"`
	Action    string            `json:"action"`
	Detail    string            `json:"detail,omitempty"`
	Method    string            `json:"method,omitempty"`
	Endpoint  string            `json:"endpoint,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
}

type AgentJobExternalLeasePreflightFilter struct {
	TargetID              string
	DesiredExecutionOwner string
	OperatorID            string
	ApprovalID            string
	JobLimit              int
	EventLimit            int
	StaleAfterSeconds     int
}

type AgentJobExternalLeaseLauncherBundleFilter struct {
	DesiredExecutionOwner string
	JobLimit              int
	EventLimit            int
	StaleAfterSeconds     int
}

type AgentJobExternalLeaseCutoverDiffFilter struct {
	DesiredExecutionOwner string
	JobLimit              int
	EventLimit            int
	StaleAfterSeconds     int
}

type AgentJobExternalLeaseLauncherBundleInputView struct {
	Name           string `json:"name"`
	Parameter      string `json:"parameter"`
	EnvironmentKey string `json:"environment_key,omitempty"`
	Required       bool   `json:"required"`
	Secret         bool   `json:"secret"`
	Detail         string `json:"detail,omitempty"`
}

type AgentJobExternalLeaseLauncherBundleView struct {
	Ready                     bool                                           `json:"ready"`
	Reason                    string                                         `json:"reason"`
	Blockers                  []string                                       `json:"blockers,omitempty"`
	DesiredExecutionOwner     string                                         `json:"desired_execution_owner"`
	RecommendedExecutionOwner string                                         `json:"recommended_execution_owner"`
	CurrentExecutionOwner     string                                         `json:"current_execution_owner"`
	Plan                      AgentJobExternalLeasePlanView                  `json:"plan"`
	ScriptPath                string                                         `json:"script_path"`
	LauncherParameters        map[string]string                              `json:"launcher_parameters,omitempty"`
	EnvironmentOverrides      map[string]string                              `json:"environment_overrides,omitempty"`
	RequiredExternalInputs    []AgentJobExternalLeaseLauncherBundleInputView `json:"required_external_inputs,omitempty"`
	VerificationSteps         []AgentJobExternalLeasePlanStep                `json:"verification_steps,omitempty"`
	Notes                     []string                                       `json:"notes,omitempty"`
	SideEffect                string                                         `json:"side_effect"`
}

type AgentJobExternalLeaseCutoverDiffItemView struct {
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	Expected   string `json:"expected,omitempty"`
	Actual     string `json:"actual,omitempty"`
	Status     string `json:"status"`
	Detail     string `json:"detail,omitempty"`
	RuntimeKey string `json:"runtime_key,omitempty"`
	Endpoint   string `json:"endpoint,omitempty"`
}

type AgentJobExternalLeaseCutoverDiffView struct {
	Ready                      bool                                       `json:"ready"`
	Reason                     string                                     `json:"reason"`
	Blockers                   []string                                   `json:"blockers,omitempty"`
	DesiredExecutionOwner      string                                     `json:"desired_execution_owner"`
	RecommendedExecutionOwner  string                                     `json:"recommended_execution_owner"`
	CurrentExecutionOwner      string                                     `json:"current_execution_owner"`
	CurrentAckOwner            string                                     `json:"current_ack_owner,omitempty"`
	ExpectedAckOwner           string                                     `json:"expected_ack_owner,omitempty"`
	CurrentQueueProvider       string                                     `json:"current_queue_provider,omitempty"`
	ExpectedQueueProvider      string                                     `json:"expected_queue_provider,omitempty"`
	CurrentQueueMode           string                                     `json:"current_queue_mode,omitempty"`
	ExpectedQueueMode          string                                     `json:"expected_queue_mode,omitempty"`
	CurrentExternalLeaseReady  bool                                       `json:"current_external_lease_ready"`
	ExpectedExternalLeaseReady bool                                       `json:"expected_external_lease_ready"`
	Bundle                     AgentJobExternalLeaseLauncherBundleView    `json:"bundle"`
	Matching                   []AgentJobExternalLeaseCutoverDiffItemView `json:"matching,omitempty"`
	Drift                      []AgentJobExternalLeaseCutoverDiffItemView `json:"drift,omitempty"`
	Notes                      []string                                   `json:"notes,omitempty"`
	SideEffect                 string                                     `json:"side_effect"`
}

type AgentJobExternalLeasePreflightView struct {
	Ready                     bool                               `json:"ready"`
	Reason                    string                             `json:"reason"`
	Blockers                  []string                           `json:"blockers,omitempty"`
	TargetKind                string                             `json:"target_kind"`
	TargetID                  string                             `json:"target_id"`
	Action                    string                             `json:"action"`
	DesiredExecutionOwner     string                             `json:"desired_execution_owner"`
	CurrentExecutionOwner     string                             `json:"current_execution_owner,omitempty"`
	RecommendedExecutionOwner string                             `json:"recommended_execution_owner,omitempty"`
	OperatorID                string                             `json:"operator_id"`
	ApprovalID                string                             `json:"approval_id"`
	Plan                      AgentJobExternalLeasePlanView      `json:"plan"`
	ControlPreflight          ControlMutationPreflightView       `json:"control_preflight"`
	SuggestedAudit            *ControlMutationSuggestedAuditView `json:"suggested_audit,omitempty"`
	Notes                     []string                           `json:"notes,omitempty"`
	SideEffect                string                             `json:"side_effect"`
}
