package all

import (
	"github.com/xtls/xray-core/maincopy/commands/all/api"
	"github.com/xtls/xray-core/maincopy/commands/all/tls"
	"github.com/xtls/xray-core/maincopy/commands/base"
)

// go:generate go run github.com/xtls/xray-core/common/errors/errorgen

func init() {
	base.RootCommand.Commands = append(
		base.RootCommand.Commands,
		api.CmdAPI,
		// cmdConvert,
		tls.CmdTLS,
		cmdUUID,
		cmdX25519,
	)
}
