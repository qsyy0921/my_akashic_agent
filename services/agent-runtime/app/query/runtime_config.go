package query

type RuntimeConfigView struct {
	Runtime     RuntimeProcessConfigView   `json:"runtime"`
	Delivery    RuntimeDeliveryConfigView  `json:"delivery"`
	Workers     RuntimeWorkerConfigView    `json:"workers"`
	Environment []RuntimeEnvVarView        `json:"environment"`
	Readiness   RuntimeConfigReadinessView `json:"readiness"`
	Notes       []string                   `json:"notes,omitempty"`
	SideEffect  string                     `json:"side_effect"`
}

type RuntimeProcessConfigView struct {
	Address       string   `json:"address"`
	AddressSource string   `json:"address_source"`
	BotIDs        []string `json:"bot_ids"`
}

type RuntimeDeliveryConfigView struct {
	TelegramChannels               []string                          `json:"telegram_channels"`
	TelegramTokenConfigured        bool                              `json:"telegram_token_configured"`
	TelegramEndpoint               string                            `json:"telegram_endpoint,omitempty"`
	OneBotEndpoints                []RuntimeOneBotEndpointConfigView `json:"onebot_endpoints"`
	OneBotExpectedChannels         []string                          `json:"onebot_expected_channels"`
	OneBotMissingChannels          []string                          `json:"onebot_missing_channels"`
	OneBotDefaultTokenConfigured   bool                              `json:"onebot_default_token_configured"`
	OneBotPerChannelTokenChannels  []string                          `json:"onebot_per_channel_token_channels"`
	OneBotReadyForHealthProbe      bool                              `json:"onebot_ready_for_health_probe"`
	OneBotReadyForDualAccountSmoke bool                              `json:"onebot_ready_for_dual_account_smoke"`
}

type RuntimeOneBotEndpointConfigView struct {
	Channel               string   `json:"channel"`
	Transport             string   `json:"transport"`
	HTTPConfigured        bool     `json:"http_configured"`
	WebSocketConfigured   bool     `json:"websocket_configured"`
	AccessTokenConfigured bool     `json:"access_token_configured"`
	Endpoint              string   `json:"endpoint,omitempty"`
	Notes                 []string `json:"notes,omitempty"`
}

type RuntimeWorkerConfigView struct {
	AgentJobRecoveryEnabled          bool `json:"agent_job_recovery_enabled"`
	OutboxDeliveryWorkerEnabled      bool `json:"outbox_delivery_worker_enabled"`
	AgentJobStrictLeaseToken         bool `json:"agent_job_strict_lease_token"`
	QueueExternalLeaseAgentJobEnable bool `json:"queue_external_lease_agent_job_enabled"`
}

type RuntimeEnvVarView struct {
	Key           string `json:"key"`
	Present       bool   `json:"present"`
	Secret        bool   `json:"secret"`
	ValueRedacted string `json:"value_redacted,omitempty"`
}

type RuntimeConfigReadinessView struct {
	DeliveryAdaptersConfigured  bool     `json:"delivery_adapters_configured"`
	OneBotConfigured            bool     `json:"onebot_configured"`
	OneBotExpectedChannelsOK    bool     `json:"onebot_expected_channels_ok"`
	OneBotHealthProbeReady      bool     `json:"onebot_health_probe_ready"`
	OneBotDualAccountSmokeReady bool     `json:"onebot_dual_account_smoke_ready"`
	Blockers                    []string `json:"blockers,omitempty"`
}
