package conf

import (
	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/transport/global"
	"github.com/xtls/xray-core/transport/internet"
)

type TransportConfig struct {
	TCPConfig  *TCPConfig  `json:"tcpSettings"`
	HTTPConfig *HTTPConfig `json:"httpSettings"`
}

// Build implements Buildable.
func (c *TransportConfig) Build() (*global.Config, error) {
	config := new(global.Config)

	if c.TCPConfig != nil {
		ts, err := c.TCPConfig.Build()
		if err != nil {
			return nil, newError("failed to build TCP config").Base(err).AtError()
		}
		config.TransportSettings = append(config.TransportSettings, &internet.TransportConfig{
			ProtocolName: "tcp",
			Settings:     serial.ToTypedMessage(ts),
		})
	}

	if c.HTTPConfig != nil {
		ts, err := c.HTTPConfig.Build()
		if err != nil {
			return nil, newError("Failed to build HTTP config.").Base(err)
		}
		config.TransportSettings = append(config.TransportSettings, &internet.TransportConfig{
			ProtocolName: "http",
			Settings:     serial.ToTypedMessage(ts),
		})
	}

	return config, nil
}
