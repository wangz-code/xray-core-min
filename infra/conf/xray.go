package conf

import (
	"encoding/json"
	"log"
	"os"
	"strings"

	"github.com/xtls/xray-core/app/dispatcher"
	"github.com/xtls/xray-core/app/proxyman"
	"github.com/xtls/xray-core/common/serial"
	core "github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/transport/internet"
)

var (
	inboundConfigLoader = NewJSONConfigLoader(ConfigCreatorCache{
		"http":  func() interface{} { return new(HTTPServerConfig) },
		"vless": func() interface{} { return new(VLessInboundConfig) },
	}, "protocol", "settings")

	outboundConfigLoader = NewJSONConfigLoader(ConfigCreatorCache{
		"freedom": func() interface{} { return new(FreedomConfig) },
		"http":    func() interface{} { return new(HTTPClientConfig) },
		"vless":   func() interface{} { return new(VLessOutboundConfig) },
	}, "protocol", "settings")

	ctllog = log.New(os.Stderr, "xctl> ", 0)
)

func toProtocolList(s []string) ([]proxyman.KnownProtocols, error) {
	kp := make([]proxyman.KnownProtocols, 0, 8)
	for _, p := range s {
		switch strings.ToLower(p) {
		case "http":
			kp = append(kp, proxyman.KnownProtocols_HTTP)
		case "https", "tls", "ssl":
			kp = append(kp, proxyman.KnownProtocols_TLS)
		default:
			return nil, newError("Unknown protocol: ", p)
		}
	}
	return kp, nil
}

type SniffingConfig struct {
	Enabled         bool        `json:"enabled"`
	DestOverride    *StringList `json:"destOverride"`
	DomainsExcluded *StringList `json:"domainsExcluded"`
	MetadataOnly    bool        `json:"metadataOnly"`
	RouteOnly       bool        `json:"routeOnly"`
}

// Build implements Buildable.
func (c *SniffingConfig) Build() (*proxyman.SniffingConfig, error) {
	var p []string
	if c.DestOverride != nil {
		for _, protocol := range *c.DestOverride {
			switch strings.ToLower(protocol) {
			case "http":
				p = append(p, "http")
			case "tls", "https", "ssl":
				p = append(p, "tls")
			case "quic":
				p = append(p, "quic")
			case "fakedns":
				p = append(p, "fakedns")
			case "fakedns+others":
				p = append(p, "fakedns+others")
			default:
				return nil, newError("unknown protocol: ", protocol)
			}
		}
	}

	var d []string
	if c.DomainsExcluded != nil {
		for _, domain := range *c.DomainsExcluded {
			d = append(d, strings.ToLower(domain))
		}
	}

	return &proxyman.SniffingConfig{
		Enabled:             c.Enabled,
		DestinationOverride: p,
		DomainsExcluded:     d,
		MetadataOnly:        c.MetadataOnly,
		RouteOnly:           c.RouteOnly,
	}, nil
}

type MuxConfig struct {
	Enabled         bool   `json:"enabled"`
	Concurrency     int16  `json:"concurrency"`
	XudpConcurrency int16  `json:"xudpConcurrency"`
	XudpProxyUDP443 string `json:"xudpProxyUDP443"`
}

// Build creates MultiplexingConfig, Concurrency < 0 completely disables mux.
func (m *MuxConfig) Build() (*proxyman.MultiplexingConfig, error) {
	switch m.XudpProxyUDP443 {
	case "":
		m.XudpProxyUDP443 = "reject"
	case "reject", "allow", "skip":
	default:
		return nil, newError(`unknown "xudpProxyUDP443": `, m.XudpProxyUDP443)
	}
	return &proxyman.MultiplexingConfig{
		Enabled:         m.Enabled,
		Concurrency:     int32(m.Concurrency),
		XudpConcurrency: int32(m.XudpConcurrency),
		XudpProxyUDP443: m.XudpProxyUDP443,
	}, nil
}

type InboundDetourAllocationConfig struct {
	Strategy    string  `json:"strategy"`
	Concurrency *uint32 `json:"concurrency"`
	RefreshMin  *uint32 `json:"refresh"`
}

// Build implements Buildable.
func (c *InboundDetourAllocationConfig) Build() (*proxyman.AllocationStrategy, error) {
	config := new(proxyman.AllocationStrategy)
	switch strings.ToLower(c.Strategy) {
	case "always":
		config.Type = proxyman.AllocationStrategy_Always
	case "random":
		config.Type = proxyman.AllocationStrategy_Random
	case "external":
		config.Type = proxyman.AllocationStrategy_External
	default:
		return nil, newError("unknown allocation strategy: ", c.Strategy)
	}
	if c.Concurrency != nil {
		config.Concurrency = &proxyman.AllocationStrategy_AllocationStrategyConcurrency{
			Value: *c.Concurrency,
		}
	}

	if c.RefreshMin != nil {
		config.Refresh = &proxyman.AllocationStrategy_AllocationStrategyRefresh{
			Value: *c.RefreshMin,
		}
	}

	return config, nil
}

type InboundDetourConfig struct {
	Protocol       string                         `json:"protocol"`
	PortList       *PortList                      `json:"port"`
	ListenOn       *Address                       `json:"listen"`
	Settings       *json.RawMessage               `json:"settings"`
	Tag            string                         `json:"tag"`
	Allocation     *InboundDetourAllocationConfig `json:"allocate"`
	StreamSetting  *StreamConfig                  `json:"streamSettings"`
	DomainOverride *StringList                    `json:"domainOverride"`
	SniffingConfig *SniffingConfig                `json:"sniffing"`
}

// Build implements Buildable.
func (c *InboundDetourConfig) Build() (*core.InboundHandlerConfig, error) {
	receiverSettings := &proxyman.ReceiverConfig{}
	receiverSettings.PortList = c.PortList.Build()
	settings := []byte("{}")
	rawConfig, err := inboundConfigLoader.LoadWithID(settings, c.Protocol)

	if err != nil {
		return nil, newError("failed to load inbound detour config.").Base(err)
	}
	ts, err := rawConfig.(Buildable).Build()
	if err != nil {
		return nil, err
	}

	return &core.InboundHandlerConfig{
		Tag:              c.Tag,
		ReceiverSettings: serial.ToTypedMessage(receiverSettings),
		ProxySettings:    serial.ToTypedMessage(ts),
	}, nil
}

type OutboundDetourConfig struct {
	Protocol      string           `json:"protocol"`
	SendThrough   *Address         `json:"sendThrough"`
	Tag           string           `json:"tag"`
	Settings      *json.RawMessage `json:"settings"`
	StreamSetting *StreamConfig    `json:"streamSettings"`
	ProxySettings *ProxyConfig     `json:"proxySettings"`
	MuxSettings   *MuxConfig       `json:"mux"`
}

func (c *OutboundDetourConfig) checkChainProxyConfig() error {
	if c.StreamSetting == nil || c.ProxySettings == nil || c.StreamSetting.SocketSettings == nil {
		return nil
	}
	if len(c.ProxySettings.Tag) > 0 && len(c.StreamSetting.SocketSettings.DialerProxy) > 0 {
		return newError("proxySettings.tag is conflicted with sockopt.dialerProxy").AtWarning()
	}
	return nil
}

// Build implements Buildable.
func (c *OutboundDetourConfig) Build() (*core.OutboundHandlerConfig, error) {
	senderSettings := &proxyman.SenderConfig{}
	if err := c.checkChainProxyConfig(); err != nil {
		return nil, err
	}

	if c.SendThrough != nil {
		address := c.SendThrough
		if address.Family().IsDomain() {
			return nil, newError("unable to send through: " + address.String())
		}
		senderSettings.Via = address.Build()
	}

	if c.StreamSetting != nil {
		ss, err := c.StreamSetting.Build()
		if err != nil {
			return nil, err
		}
		senderSettings.StreamSettings = ss
	}

	if c.ProxySettings != nil {
		ps, err := c.ProxySettings.Build()
		if err != nil {
			return nil, newError("invalid outbound detour proxy settings.").Base(err)
		}
		if ps.TransportLayerProxy {
			if senderSettings.StreamSettings != nil {
				if senderSettings.StreamSettings.SocketSettings != nil {
					senderSettings.StreamSettings.SocketSettings.DialerProxy = ps.Tag
				} else {
					senderSettings.StreamSettings.SocketSettings = &internet.SocketConfig{DialerProxy: ps.Tag}
				}
			} else {
				senderSettings.StreamSettings = &internet.StreamConfig{SocketSettings: &internet.SocketConfig{DialerProxy: ps.Tag}}
			}
			ps = nil
		}
		senderSettings.ProxySettings = ps
	}

	if c.MuxSettings != nil {
		ms, err := c.MuxSettings.Build()
		if err != nil {
			return nil, newError("failed to build Mux config.").Base(err)
		}
		senderSettings.MultiplexSettings = ms
	}

	settings := []byte("{}")
	if c.Settings != nil {
		settings = ([]byte)(*c.Settings)
	}
	rawConfig, err := outboundConfigLoader.LoadWithID(settings, c.Protocol)
	if err != nil {
		return nil, newError("failed to parse to outbound detour config.").Base(err)
	}
	ts, err := rawConfig.(Buildable).Build()
	if err != nil {
		return nil, err
	}

	return &core.OutboundHandlerConfig{
		SenderSettings: serial.ToTypedMessage(senderSettings),
		Tag:            c.Tag,
		ProxySettings:  serial.ToTypedMessage(ts),
	}, nil
}

type Config struct {
	// Port of this Point server.
	// Deprecated: Port exists for historical compatibility
	// and should not be used.
	Port uint16 `json:"port"`

	// Deprecated: InboundConfig exists for historical compatibility
	// and should not be used.
	InboundConfig *InboundDetourConfig `json:"inbound"`

	// Deprecated: OutboundConfig exists for historical compatibility
	// and should not be used.
	OutboundConfig *OutboundDetourConfig `json:"outbound"`

	// Deprecated: InboundDetours exists for historical compatibility
	// and should not be used.
	InboundDetours []InboundDetourConfig `json:"inboundDetour"`

	// Deprecated: OutboundDetours exists for historical compatibility
	// and should not be used.
	OutboundDetours []OutboundDetourConfig `json:"outboundDetour"`

	LogConfig       *LogConfig             `json:"log"`
	RouterConfig    *RouterConfig          `json:"routing"`
	InboundConfigs  []InboundDetourConfig  `json:"inbounds"`
	OutboundConfigs []OutboundDetourConfig `json:"outbounds"`
	Transport       *TransportConfig       `json:"transport"`
	// Policy          *PolicyConfig          `json:"policy"`
	Observatory *ObservatoryConfig `json:"observatory"`
}

func (c *Config) findInboundTag(tag string) int {
	found := -1
	for idx, ib := range c.InboundConfigs {
		if ib.Tag == tag {
			found = idx
			break
		}
	}
	return found
}

func (c *Config) findOutboundTag(tag string) int {
	found := -1
	for idx, ob := range c.OutboundConfigs {
		if ob.Tag == tag {
			found = idx
			break
		}
	}
	return found
}

// Override method accepts another Config overrides the current attribute
func (c *Config) Override(o *Config, fn string) {
	// only process the non-deprecated members

	if o.LogConfig != nil {
		c.LogConfig = o.LogConfig
	}
	if o.RouterConfig != nil {
		c.RouterConfig = o.RouterConfig
	}
	if o.Transport != nil {
		c.Transport = o.Transport
	}
	if o.Observatory != nil {
		c.Observatory = o.Observatory
	}

	// deprecated attrs... keep them for now
	if o.InboundConfig != nil {
		c.InboundConfig = o.InboundConfig
	}
	if o.OutboundConfig != nil {
		c.OutboundConfig = o.OutboundConfig
	}
	if o.InboundDetours != nil {
		c.InboundDetours = o.InboundDetours
	}
	if o.OutboundDetours != nil {
		c.OutboundDetours = o.OutboundDetours
	}
	// deprecated attrs

	// update the Inbound in slice if the only one in override config has same tag
	if len(o.InboundConfigs) > 0 {
		for i := range o.InboundConfigs {
			if idx := c.findInboundTag(o.InboundConfigs[i].Tag); idx > -1 {
				c.InboundConfigs[idx] = o.InboundConfigs[i]
				newError("[", fn, "] updated inbound with tag: ", o.InboundConfigs[i].Tag).AtInfo().WriteToLog()

			} else {
				c.InboundConfigs = append(c.InboundConfigs, o.InboundConfigs[i])
				newError("[", fn, "] appended inbound with tag: ", o.InboundConfigs[i].Tag).AtInfo().WriteToLog()
			}

		}
	}

	// update the Outbound in slice if the only one in override config has same tag
	if len(o.OutboundConfigs) > 0 {
		outboundPrepends := []OutboundDetourConfig{}
		for i := range o.OutboundConfigs {
			if idx := c.findOutboundTag(o.OutboundConfigs[i].Tag); idx > -1 {
				c.OutboundConfigs[idx] = o.OutboundConfigs[i]
				newError("[", fn, "] updated outbound with tag: ", o.OutboundConfigs[i].Tag).AtInfo().WriteToLog()
			} else {
				if strings.Contains(strings.ToLower(fn), "tail") {
					c.OutboundConfigs = append(c.OutboundConfigs, o.OutboundConfigs[i])
					newError("[", fn, "] appended outbound with tag: ", o.OutboundConfigs[i].Tag).AtInfo().WriteToLog()
				} else {
					outboundPrepends = append(outboundPrepends, o.OutboundConfigs[i])
					newError("[", fn, "] prepend outbound with tag: ", o.OutboundConfigs[i].Tag).AtInfo().WriteToLog()
				}
			}
		}
		if !strings.Contains(strings.ToLower(fn), "tail") && len(outboundPrepends) > 0 {
			c.OutboundConfigs = append(outboundPrepends, c.OutboundConfigs...)
		}
	}
}

func applyTransportConfig(s *StreamConfig, t *TransportConfig) {
	if s.TCPSettings == nil {
		s.TCPSettings = t.TCPConfig
	}

}

// Build implements Buildable.
func (c *Config) Build() (*core.Config, error) {
	if err := PostProcessConfigureFile(c); err != nil {
		return nil, err
	}

	config := &core.Config{
		App: []*serial.TypedMessage{
			serial.ToTypedMessage(&dispatcher.Config{}),
			serial.ToTypedMessage(&proxyman.InboundConfig{}),
			serial.ToTypedMessage(&proxyman.OutboundConfig{}),
		},
	}

	var logConfMsg *serial.TypedMessage
	if c.LogConfig != nil {
		logConfMsg = serial.ToTypedMessage(c.LogConfig.Build())
	} else {
		logConfMsg = serial.ToTypedMessage(DefaultLogConfig())
	}
	// let logger module be the first App to start,
	// so that other modules could print log during initiating
	config.App = append([]*serial.TypedMessage{logConfMsg}, config.App...)

	if c.RouterConfig != nil {
		routerConfig, err := c.RouterConfig.Build()
		if err != nil {
			return nil, err
		}
		config.App = append(config.App, serial.ToTypedMessage(routerConfig))
	}

	var inbounds []InboundDetourConfig
	inbounds = append(inbounds, c.InboundConfigs...)
	rawInboundConfig := inbounds[0]
	ic, err := rawInboundConfig.Build()
	if err != nil {
		return nil, err
	}
	config.Inbound = append(config.Inbound, ic)

	var outbounds []OutboundDetourConfig

	if c.OutboundConfig != nil {
		outbounds = append(outbounds, *c.OutboundConfig)
	}

	if len(c.OutboundDetours) > 0 {
		outbounds = append(outbounds, c.OutboundDetours...)
	}

	if len(c.OutboundConfigs) > 0 {
		outbounds = append(outbounds, c.OutboundConfigs...)
	}

	for _, rawOutboundConfig := range outbounds {
		if c.Transport != nil {
			if rawOutboundConfig.StreamSetting == nil {
				rawOutboundConfig.StreamSetting = &StreamConfig{}
			}
			applyTransportConfig(rawOutboundConfig.StreamSetting, c.Transport)
		}
		oc, err := rawOutboundConfig.Build()
		if err != nil {
			return nil, err
		}
		config.Outbound = append(config.Outbound, oc)
	}

	return config, nil
}
