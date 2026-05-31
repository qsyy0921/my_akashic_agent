package model

import "strings"

func (c ChannelRef) AccountKey() string {
	return strings.TrimSpace(string(c.Kind)) + ":" + strings.TrimSpace(c.AccountID)
}
