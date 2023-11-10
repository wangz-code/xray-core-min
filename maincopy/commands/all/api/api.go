package api

import (
	"github.com/xtls/xray-core/maincopy/commands/base"
)

// CmdAPI calls an API in an Xray process
var CmdAPI = &base.Command{
	UsageLine: "{{.Exec}} api",
	Short:     "Call an API in an Xray process",
	Long: `{{.Exec}} {{.LongName}} provides tools to manipulate Xray via its API.
`,
	Commands: []*base.Command{
		cmdRestartLogger,
		cmdAddInbounds,
		cmdAddOutbounds,
		cmdRemoveInbounds,
		cmdRemoveOutbounds,
	},
}
