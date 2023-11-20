package socks

import (
	"context"
	"io"
	"time"

	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/common/buf"
	"github.com/xtls/xray-core/common/log"
	"github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/common/session"
	"github.com/xtls/xray-core/common/signal"
	"github.com/xtls/xray-core/common/task"
	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/features"
	"github.com/xtls/xray-core/features/policy"
	"github.com/xtls/xray-core/features/routing"
	"github.com/xtls/xray-core/transport/internet/stat"
)

// Server is a SOCKS 5 proxy server
type Server struct {
	config        *ServerConfig
	policyManager policy.Manager
	cone          bool
}

// NewServer creates a new Server object.
func NewServer(ctx context.Context, config *ServerConfig) (*Server, error) {
	v := core.MustFromContext(ctx)
	s := &Server{
		config:        config,
		policyManager: v.GetFeature(policy.ManagerType()).(policy.Manager),
		cone:          ctx.Value("cone").(bool),
	}
	return s, nil
}

func (s *Server) policy() policy.Session {
	config := s.config
	p := s.policyManager.ForLevel(config.UserLevel)
	if config.Timeout > 0 {
		features.PrintDeprecatedFeatureWarning("Socks timeout")
	}
	if config.Timeout > 0 && config.UserLevel == 0 {
		p.Timeouts.ConnectionIdle = time.Duration(config.Timeout) * time.Second
	}
	return p
}

// Network implements proxy.Inbound.
func (s *Server) Network() []net.Network {
	list := []net.Network{net.Network_TCP}
	if s.config.UdpEnabled {
		list = append(list, net.Network_UDP)
	}
	return list
}

// Process implements proxy.Inbound.
func (s *Server) Process(ctx context.Context, network net.Network, conn stat.Connection, dispatcher routing.Dispatcher) error {
	inbound := session.InboundFromContext(ctx)
	inbound.Name = "socks"
	inbound.SetCanSpliceCopy(2)
	inbound.User = &protocol.MemoryUser{
		Level: s.config.UserLevel,
	}

	switch network {
	case net.Network_TCP:
		return s.processTCP(ctx, conn, dispatcher)
	default:
		return newError("unknown network: ", network)
	}
}

func (s *Server) processTCP(ctx context.Context, conn stat.Connection, dispatcher routing.Dispatcher) error {
	plcy := s.policy()
	if err := conn.SetReadDeadline(time.Now().Add(plcy.Timeouts.Handshake)); err != nil {
		newError("failed to set deadline").Base(err).WriteToLog(session.ExportIDToError(ctx))
	}

	inbound := session.InboundFromContext(ctx)
	if inbound == nil || !inbound.Gateway.IsValid() {
		return newError("inbound gateway not specified")
	}

	svrSession := &ServerSession{
		config:       s.config,
		address:      inbound.Gateway.Address,
		port:         inbound.Gateway.Port,
		localAddress: net.IPAddress(conn.LocalAddr().(*net.TCPAddr).IP),
	}

	reader := &buf.BufferedReader{Reader: buf.NewReader(conn)}
	request, err := svrSession.Handshake(reader, conn)
	if err != nil {
		if inbound != nil && inbound.Source.IsValid() {
			log.Record(&log.AccessMessage{
				From:   inbound.Source,
				To:     "",
				Status: log.AccessRejected,
				Reason: err,
			})
		}
		return newError("failed to read request").Base(err)
	}
	if request.User != nil {
		inbound.User.Email = request.User.Email
	}

	if err := conn.SetReadDeadline(time.Time{}); err != nil {
		newError("failed to clear deadline").Base(err).WriteToLog(session.ExportIDToError(ctx))
	}

	if request.Command == protocol.RequestCommandTCP {
		dest := request.Destination()
		newError("TCP Connect request to ", dest).WriteToLog(session.ExportIDToError(ctx))
		if inbound != nil && inbound.Source.IsValid() {
			ctx = log.ContextWithAccessMessage(ctx, &log.AccessMessage{
				From:   inbound.Source,
				To:     dest,
				Status: log.AccessAccepted,
				Reason: "",
			})
		}

		return s.transport(ctx, reader, conn, dest, dispatcher, inbound)
	}

	if request.Command == protocol.RequestCommandUDP {
		return s.handleUDP(conn)
	}

	return nil
}

func (*Server) handleUDP(c io.Reader) error {
	// The TCP connection closes after this method returns. We need to wait until
	// the client closes it.
	return common.Error2(io.Copy(buf.DiscardBytes, c))
}

func (s *Server) transport(ctx context.Context, reader io.Reader, writer io.Writer, dest net.Destination, dispatcher routing.Dispatcher, inbound *session.Inbound) error {
	ctx, cancel := context.WithCancel(ctx)
	timer := signal.CancelAfterInactivity(ctx, cancel, s.policy().Timeouts.ConnectionIdle)

	if inbound != nil {
		inbound.Timer = timer
	}

	plcy := s.policy()
	ctx = policy.ContextWithBufferPolicy(ctx, plcy.Buffer)
	link, err := dispatcher.Dispatch(ctx, dest)
	if err != nil {
		return err
	}

	requestDone := func() error {
		defer timer.SetTimeout(plcy.Timeouts.DownlinkOnly)
		if err := buf.Copy(buf.NewReader(reader), link.Writer, buf.UpdateActivity(timer)); err != nil {
			return newError("failed to transport all TCP request").Base(err)
		}

		return nil
	}

	responseDone := func() error {
		defer timer.SetTimeout(plcy.Timeouts.UplinkOnly)

		v2writer := buf.NewWriter(writer)
		if err := buf.Copy(link.Reader, v2writer, buf.UpdateActivity(timer)); err != nil {
			return newError("failed to transport all TCP response").Base(err)
		}

		return nil
	}

	requestDonePost := task.OnSuccess(requestDone, task.Close(link.Writer))
	if err := task.Run(ctx, requestDonePost, responseDone); err != nil {
		common.Interrupt(link.Reader)
		common.Interrupt(link.Writer)
		return newError("connection ends").Base(err)
	}

	return nil
}

func init() {
	common.Must(common.RegisterConfig((*ServerConfig)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		return NewServer(ctx, config.(*ServerConfig))
	}))
}
