package query

type DeliveryAdapterDiagnosticsView struct {
	Provider              string   `json:"provider"`
	Channel               string   `json:"channel"`
	Transport             string   `json:"transport"`
	Enabled               bool     `json:"enabled"`
	EndpointConfigured    bool     `json:"endpoint_configured"`
	AccessTokenConfigured bool     `json:"access_token_configured"`
	Endpoint              string   `json:"endpoint,omitempty"`
	Notes                 []string `json:"notes,omitempty"`
}
