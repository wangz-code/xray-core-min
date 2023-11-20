package dns

//go:generate go run github.com/xtls/xray-core/common/errors/errorgen

import (
	"github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/features/routing"
)

// ResolvableContext is an implementation of routing.Context, with domain resolving capability.
type ResolvableContext struct {
	routing.Context
	resolvedIPs []net.IP
}

// GetTargetIPs overrides original routing.Context's implementation.
func (ctx *ResolvableContext) GetTargetIPs() []net.IP {
	if len(ctx.resolvedIPs) > 0 {
		return ctx.resolvedIPs
	}

	if domain := ctx.GetTargetDomain(); len(domain) != 0 {

		newError("resolve ip for ", domain).Base(newError("错误")).WriteToLog()
	}

	if ips := ctx.Context.GetTargetIPs(); len(ips) != 0 {
		return ips
	}

	return nil
}

// ContextWithDNSClient creates a new routing context with domain resolving capability.
// Resolved domain IPs can be retrieved by GetTargetIPs().
func ContextWithDNSClient(ctx routing.Context) routing.Context {
	return &ResolvableContext{Context: ctx}
}
