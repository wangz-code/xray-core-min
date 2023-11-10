package conf

import (
	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/transport/global"
	"github.com/xtls/xray-core/transport/internet"
)

type TransportConfig struct {
	TCPConfig *TCPConfig `json:"tcpSettings"`
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

	return config, nil
}
