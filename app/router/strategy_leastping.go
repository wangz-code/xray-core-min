package router

import (
	"context"
)

type LeastPingStrategy struct {
	ctx context.Context
}

func (l *LeastPingStrategy) InjectContext(ctx context.Context) {
	l.ctx = ctx
}

func (l *LeastPingStrategy) PickOutbound(strings []string) string {

	// No way to understand observeReport
	return ""
}

type outboundList []string

func (o outboundList) contains(name string) bool {
	for _, v := range o {
		if v == name {
			return true
		}
	}
	return false
}
