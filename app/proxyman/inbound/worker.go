package inbound

import (
	"context"

	"github.com/xtls/xray-core/app/proxyman"
	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/common/session"
	"github.com/xtls/xray-core/features/routing"
	"github.com/xtls/xray-core/features/stats"
	"github.com/xtls/xray-core/proxy"
	"github.com/xtls/xray-core/transport/internet"
	"github.com/xtls/xray-core/transport/internet/stat"
	"github.com/xtls/xray-core/transport/internet/tcp"
)

type worker interface {
	Start() error
	Close() error
	Port() net.Port
	Proxy() proxy.Inbound
}

type tcpWorker struct {
	address         net.Address
	port            net.Port
	proxy           proxy.Inbound
	stream          *internet.MemoryStreamConfig
	recvOrigDest    bool
	tag             string
	dispatcher      routing.Dispatcher
	sniffingConfig  *proxyman.SniffingConfig
	uplinkCounter   stats.Counter
	downlinkCounter stats.Counter

	hub internet.Listener

	ctx context.Context
}

func getTProxyType(s *internet.MemoryStreamConfig) internet.SocketConfig_TProxyMode {
	if s == nil || s.SocketSettings == nil {
		return internet.SocketConfig_Off
	}
	return s.SocketSettings.Tproxy
}

func (w *tcpWorker) callback(conn stat.Connection) {
	ctx, cancel := context.WithCancel(w.ctx)
	sid := session.NewID()
	ctx = session.ContextWithID(ctx, sid)

	var outbound = &session.Outbound{}
	if w.recvOrigDest {
		var dest net.Destination
		switch getTProxyType(w.stream) {
		case internet.SocketConfig_Redirect:
			d, err := tcp.GetOriginalDestination(conn)
			if err != nil {
				newError("failed to get original destination").Base(err).WriteToLog(session.ExportIDToError(ctx))
			} else {
				dest = d
			}
		case internet.SocketConfig_TProxy:
			dest = net.DestinationFromAddr(conn.LocalAddr())
		}
		if dest.IsValid() {
			outbound.Target = dest
		}
	}
	ctx = session.ContextWithOutbound(ctx, outbound)

	if w.uplinkCounter != nil || w.downlinkCounter != nil {
		conn = &stat.CounterConnection{
			Connection:   conn,
			ReadCounter:  w.uplinkCounter,
			WriteCounter: w.downlinkCounter,
		}
	}
	ctx = session.ContextWithInbound(ctx, &session.Inbound{
		Source:  net.DestinationFromAddr(conn.RemoteAddr()),
		Gateway: net.TCPDestination(w.address, w.port),
		Tag:     w.tag,
		Conn:    conn,
	})

	content := new(session.Content)
	if w.sniffingConfig != nil {
		content.SniffingRequest.Enabled = w.sniffingConfig.Enabled
		content.SniffingRequest.OverrideDestinationForProtocol = w.sniffingConfig.DestinationOverride
		content.SniffingRequest.ExcludeForDomain = w.sniffingConfig.DomainsExcluded
		content.SniffingRequest.MetadataOnly = w.sniffingConfig.MetadataOnly
		content.SniffingRequest.RouteOnly = w.sniffingConfig.RouteOnly
	}
	ctx = session.ContextWithContent(ctx, content)

	if err := w.proxy.Process(ctx, net.Network_TCP, conn, w.dispatcher); err != nil {
		newError("connection ends").Base(err).WriteToLog(session.ExportIDToError(ctx))
	}
	cancel()
	conn.Close()
}

func (w *tcpWorker) Proxy() proxy.Inbound {
	return w.proxy
}

func (w *tcpWorker) Start() error {
	ctx := context.Background()
	hub, err := internet.ListenTCP(ctx, w.address, w.port, w.stream, func(conn stat.Connection) {
		go w.callback(conn)
	})
	if err != nil {
		return newError("failed to listen TCP on ", w.port).AtWarning().Base(err)
	}
	w.hub = hub
	return nil
}

func (w *tcpWorker) Close() error {
	var errors []interface{}
	if w.hub != nil {
		if err := common.Close(w.hub); err != nil {
			errors = append(errors, err)
		}
		if err := common.Close(w.proxy); err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		return newError("failed to close all resources").Base(newError(serial.Concat(errors...)))
	}

	return nil
}

func (w *tcpWorker) Port() net.Port {
	return w.port
}

type dsWorker struct {
	address         net.Address
	proxy           proxy.Inbound
	stream          *internet.MemoryStreamConfig
	tag             string
	dispatcher      routing.Dispatcher
	sniffingConfig  *proxyman.SniffingConfig
	uplinkCounter   stats.Counter
	downlinkCounter stats.Counter

	hub internet.Listener

	ctx context.Context
}

func (w *dsWorker) callback(conn stat.Connection) {
	ctx, cancel := context.WithCancel(w.ctx)
	sid := session.NewID()
	ctx = session.ContextWithID(ctx, sid)

	if w.uplinkCounter != nil || w.downlinkCounter != nil {
		conn = &stat.CounterConnection{
			Connection:   conn,
			ReadCounter:  w.uplinkCounter,
			WriteCounter: w.downlinkCounter,
		}
	}
	ctx = session.ContextWithInbound(ctx, &session.Inbound{
		Source:  net.DestinationFromAddr(conn.RemoteAddr()),
		Gateway: net.UnixDestination(w.address),
		Tag:     w.tag,
		Conn:    conn,
	})

	content := new(session.Content)
	if w.sniffingConfig != nil {
		content.SniffingRequest.Enabled = w.sniffingConfig.Enabled
		content.SniffingRequest.OverrideDestinationForProtocol = w.sniffingConfig.DestinationOverride
		content.SniffingRequest.ExcludeForDomain = w.sniffingConfig.DomainsExcluded
		content.SniffingRequest.MetadataOnly = w.sniffingConfig.MetadataOnly
		content.SniffingRequest.RouteOnly = w.sniffingConfig.RouteOnly
	}
	ctx = session.ContextWithContent(ctx, content)

	if err := w.proxy.Process(ctx, net.Network_UNIX, conn, w.dispatcher); err != nil {
		newError("connection ends").Base(err).WriteToLog(session.ExportIDToError(ctx))
	}
	cancel()
	if err := conn.Close(); err != nil {
		newError("failed to close connection").Base(err).WriteToLog(session.ExportIDToError(ctx))
	}
}

func (w *dsWorker) Proxy() proxy.Inbound {
	return w.proxy
}

func (w *dsWorker) Port() net.Port {
	return net.Port(0)
}

func (w *dsWorker) Start() error {
	ctx := context.Background()
	hub, err := internet.ListenUnix(ctx, w.address, w.stream, func(conn stat.Connection) {
		go w.callback(conn)
	})
	if err != nil {
		return newError("failed to listen Unix Domain Socket on ", w.address).AtWarning().Base(err)
	}
	w.hub = hub
	return nil
}

func (w *dsWorker) Close() error {
	var errors []interface{}
	if w.hub != nil {
		if err := common.Close(w.hub); err != nil {
			errors = append(errors, err)
		}
		if err := common.Close(w.proxy); err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		return newError("failed to close all resources").Base(newError(serial.Concat(errors...)))
	}

	return nil
}
