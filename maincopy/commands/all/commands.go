package all

import (
	"github.com/xtls/xray-core/maincopy/commands/base"
)

// go:generate go run github.com/xtls/xray-core/common/errors/errorgen

func init() {
	base.RootCommand.Commands = append(
		base.RootCommand.Commands,
	)
}
