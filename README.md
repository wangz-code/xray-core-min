### 尝试精简 xray-core

https://github.com/XTLS/Xray-core/issues/1880#issuecomment-1491614281 看作者说想要精简, 我翻了好几遍好像没找到相关的项目, 俗话说: 求人不如求只因

小米 4A 千兆版刚刷好的 openwrt 运行不了 xray 实在是难受, (free 只有 7.6M upx 后的 xray 还有 8.4M) 遂想要精简下 xray-core 仅保留用到的, 其他都删除

### 修改了 main.go 位置, 主要是为了方便 air 即时编译

main.go 放倒外面其他的只要 config.json 中没有使用的全都删除,写的慢但是删的是真不慢, blackhole 和 DNS 我都删了, 应该还能再删, 这里有些文件夹我不确定有什么用,官方文档也没说明 难受

仅保留:

-   socks
-   http
-   freedom
-   vless

> 删了这么多竟然还能运行, 我震惊一百年!

### Linux / macOS

```bash
go build -o xray -trimpath -ldflags "-s -w -buildid=" ./main
go build -o xray -trimpath -ldflags "-s -w -buildid=" main.go


# 在mac下编译后xray大概21M
# upx 之后xray大概  9.1M
```

### openwrt mipsel_24kc

真机好像只有 softfloat 这种才能运行, mac m1 下无法交叉编译, 只能在 linux 上才能编译,测试 centos8 可以 build

```bash
CGO_ENABLED=0 GOARCH=mipsle GOMIPS=softfloat go build -o xray -trimpath -ldflags "-s -w -buildid=" ./main

```

### 测试能够运行的 config.json

```json
{
	"inbounds": [
		{
			"listen": "127.0.0.1",
			"port": 10800,
			"protocol": "socks",
			"settings": {
				"udp": true
			},
			"tag": "socks-in"
		},
		{
			"listen": "127.0.0.1",
			"port": 10801,
			"protocol": "http",
			"tag": "http-in"
		}
	],
	"outbounds": [
		{
			"protocol": "freedom",
			"tag": "direct"
		},
		{
			"protocol": "vless",
			"settings": {
				"vnext": [
					{
						"address": "x.x.x.x",
						"port": 443,
						"users": [
							{
								"alterId": 64,
								"encryption": "none",
								"flow": "xtls-rprx-vision",
								"id": "abababbaba-abababba-abab-abbab-ababba",
								"level": 1,
								"security": "none"
							}
						]
					}
				]
			},
			"streamSettings": {
				"network": "tcp",
				"security": "tls",
				"tlsSettings": {
					"allowInsecure": false,
					"allowInsecureCiphers": false,
					"alpn": ["h2"],
					"fingerprint": "chrome",
					"serverName": "www.domain.xyz"
				}
			},
			"tag": "proxy"
		}
	],
	"routing": {
		"domainMatcher": "hybrid",
		"domainStrategy": "AsIs",
		"rules": [
			{
				"domain": ["domain:google.com", "domain:google.com.hk"],
				"outboundTag": "proxy",
				"type": "field"
			},
			{
				"domain": ["domain:taobao.com", "domain:jd.com"],
				"outboundTag": "direct",
				"type": "field"
			}
		]
	}
}
```
