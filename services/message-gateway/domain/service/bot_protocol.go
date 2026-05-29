package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"

	"github.com/kachofugetsu09/akashic-agent/services/message-gateway/domain/model"
)

const botProtocolPrefix = "[[akashic:bot"

var botProtocolPattern = regexp.MustCompile(`\[\[akashic:bot from=([0-9A-Za-z_\-]+) nonce=([0-9A-Za-z_\-]+) hop=([0-9]+)\]\]`)

func EncodeBotProtocolTag(protocol model.BotProtocol) string {
	return fmt.Sprintf("%s from=%s nonce=%s hop=%d]]", botProtocolPrefix, protocol.FromBotID, protocol.Nonce, protocol.Hop)
}

func ParseBotProtocolTag(content string) (model.BotProtocol, bool) {
	match := botProtocolPattern.FindStringSubmatch(content)
	if len(match) != 4 {
		return model.BotProtocol{}, false
	}

	hop, err := strconv.Atoi(match[3])
	if err != nil {
		return model.BotProtocol{}, false
	}

	return model.BotProtocol{
		FromBotID: match[1],
		Nonce:     match[2],
		Hop:       hop,
	}, true
}

func PrependBotProtocol(content string, protocol model.BotProtocol) string {
	return EncodeBotProtocolTag(protocol) + "\n" + content
}

func NewNonce() string {
	var buf [12]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return ""
	}
	return hex.EncodeToString(buf[:])
}
