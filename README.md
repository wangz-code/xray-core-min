### 尝试精简

开始吧

### Linux / macOS

```bash
go build -o xray -trimpath -ldflags "-s -w -buildid=" ./main
```

### openwrt32

```bash
CGO_ENABLED=0 GOARCH=mips GOMIPS=softfloat go build -o xray -trimpath -ldflags "-s -w -buildid=" ./main

```
