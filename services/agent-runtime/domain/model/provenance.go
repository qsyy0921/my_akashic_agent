package model

type ProvenanceType string

const (
	ProvenanceHuman    ProvenanceType = "human"
	ProvenanceSelfEcho ProvenanceType = "self_echo"
	ProvenancePeerBot  ProvenanceType = "peer_bot"
	ProvenanceSystem   ProvenanceType = "system"
)

type Provenance struct {
	Type           ProvenanceType
	FromBotID      string
	ContentHash    string
	Nonce          string
	Hop            int
	HasProtocolTag bool
}

type BotProtocol struct {
	FromBotID string
	Nonce     string
	Hop       int
}

