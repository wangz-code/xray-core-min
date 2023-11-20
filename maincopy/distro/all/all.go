package all

import (
	// The following are necessary as they register handlers in their init functions.

	// Mandatory features. Can't remove unless there are replacements.
	_ "github.com/xtls/xray-core/app/dispatcher"
	_ "github.com/xtls/xray-core/app/proxyman/inbound"
	_ "github.com/xtls/xray-core/app/proxyman/outbound"

	// Default commander and all its services. This is an optional feature.
	_ "github.com/xtls/xray-core/app/commander"
	_ "github.com/xtls/xray-core/app/log/command"

	_ "github.com/xtls/xray-core/app/proxyman/command"

	// Developer preview services
	_ "github.com/xtls/xray-core/app/observatory/command"

	// Other optional features.
	_ "github.com/xtls/xray-core/app/log"

	_ "github.com/xtls/xray-core/app/router"

	// Fix dependency cycle caused by core import in internet package
	_ "github.com/xtls/xray-core/transport/internet/tagged/taggedimpl"

	// Developer preview features
	_ "github.com/xtls/xray-core/app/observatory"

	// Inbound and outbound proxies.
	_ "github.com/xtls/xray-core/proxy/freedom"
	_ "github.com/xtls/xray-core/proxy/http"
	_ "github.com/xtls/xray-core/proxy/socks"
	_ "github.com/xtls/xray-core/proxy/vless/inbound"
	_ "github.com/xtls/xray-core/proxy/vless/outbound"

	// _ "github.com/xtls/xray-core/transport/internet/http"

	_ "github.com/xtls/xray-core/transport/internet/tcp"
	_ "github.com/xtls/xray-core/transport/internet/tls"

	// Transport headers
	_ "github.com/xtls/xray-core/transport/internet/headers/http"
	_ "github.com/xtls/xray-core/transport/internet/headers/noop"
	_ "github.com/xtls/xray-core/transport/internet/headers/srtp"
	_ "github.com/xtls/xray-core/transport/internet/headers/tls"
	_ "github.com/xtls/xray-core/transport/internet/headers/utp"
	_ "github.com/xtls/xray-core/transport/internet/headers/wechat"
	_ "github.com/xtls/xray-core/transport/internet/headers/wireguard"

	// JSON & TOML & YAML
	_ "github.com/xtls/xray-core/maincopy/json"

	// Load config from file or http(s)
	_ "github.com/xtls/xray-core/maincopy/confloader/external"

	// Commands
	_ "github.com/xtls/xray-core/maincopy/commands/all"
)
