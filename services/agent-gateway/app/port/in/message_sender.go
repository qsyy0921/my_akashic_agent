package inport

import (
	"context"

	"github.com/kachofugetsu09/akashic-agent/services/agent-gateway/app/command"
)

type MessageSender interface {
	Send(ctx context.Context, cmd command.SendMessageCommand) error
}
