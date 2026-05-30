package query

type DeliverySmokeReadinessView struct {
	Ready      bool                             `json:"ready"`
	Reason     string                           `json:"reason"`
	Cases      []DeliverySmokeCaseReadinessView `json:"cases"`
	Totals     map[string]int                   `json:"totals"`
	Blockers   []string                         `json:"blockers,omitempty"`
	Attributes map[string]string                `json:"attributes,omitempty"`
	Notes      []string                         `json:"notes,omitempty"`
	SideEffect string                           `json:"side_effect"`
}

type DeliverySmokeCaseReadinessView struct {
	Name            string                   `json:"name"`
	Ready           bool                     `json:"ready"`
	Reason          string                   `json:"reason"`
	MissingChannels []string                 `json:"missing_channels,omitempty"`
	Plan            DeliveryDispatchPlanView `json:"plan"`
	Attributes      map[string]string        `json:"attributes,omitempty"`
}
