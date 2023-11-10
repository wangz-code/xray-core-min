package commander

//go:generate go run github.com/xtls/xray-core/common/errors/errorgen
// 在构建时通过该命令自动生成错误相关的代码

import (
	"context"
	"net"
	"sync"

	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/common/signal/done"
	core "github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/features/outbound"
	"google.golang.org/grpc"
)

// Commander是一个Xray特性，为外部客户端提供gRPC方法。
type Commander struct {
	sync.Mutex
	server   *grpc.Server
	services []Service
	ohm      outbound.Manager
	tag      string
}

// NewCommander基于给定的配置创建一个新的Commander。
func NewCommander(ctx context.Context, config *Config) (*Commander, error) {
	// 创建Commander实例，并设置标签
	c := &Commander{
		tag: config.Tag,
	}

	// 使用核心功能要求外部注入outbound.Manager
	common.Must(core.RequireFeatures(ctx, func(om outbound.Manager) {
		c.ohm = om
	}))

	// 循环处理配置中的所有服务
	for _, rawConfig := range config.Service {
		// 获取服务实例的配置信息
		config, err := rawConfig.GetInstance()
		if err != nil {
			return nil, err
		}
		// 创建服务实例
		rawService, err := common.CreateObject(ctx, config)
		if err != nil {
			return nil, err
		}
		// 将服务实例转换为Service接口类型，并添加到services列表中
		service, ok := rawService.(Service)
		if !ok {
			return nil, newError("not a Service.") // 不是Service类型的错误
		}
		c.services = append(c.services, service)
	}

	return c, nil
}

// Type实现common.HasType接口，用于返回Commander的类型信息。
func (c *Commander) Type() interface{} {
	return (*Commander)(nil)
}

// Start实现common.Runnable接口。
func (c *Commander) Start() error {
	c.Lock()
	c.server = grpc.NewServer()

	// 注册所有服务到gRPC服务器
	for _, service := range c.services {
		service.Register(c.server)
	}
	c.Unlock()

	// 创建一个OutboundListener作为gRPC服务器的监听器
	listener := &OutboundListener{
		buffer: make(chan net.Conn, 4),
		done:   done.New(),
	}

	go func() {
		// 启动gRPC服务器，并通过listener监听连接请求
		if err := c.server.Serve(listener); err != nil {
			newError("failed to start grpc server").Base(err).AtError().WriteToLog()
		}
	}()

	// 从outbound.Manager中删除现有的处理程序
	if err := c.ohm.RemoveHandler(context.Background(), c.tag); err != nil {
		newError("failed to remove existing handler").WriteToLog()
	}

	// 添加Commander作为新的处理程序，将连接请求传递给OutboundListener
	return c.ohm.AddHandler(context.Background(), &Outbound{
		tag:      c.tag,
		listener: listener,
	})
}

// Close实现common.Closable接口。
func (c *Commander) Close() error {
	c.Lock()
	defer c.Unlock()

	if c.server != nil {
		// 停止gRPC服务器
		c.server.Stop()
		c.server = nil
	}

	return nil
}

func init() {
	// 注册配置，当配置信息被加载时，会创建一个Commander实例
	common.Must(common.RegisterConfig((*Config)(nil), func(ctx context.Context, cfg interface{}) (interface{}, error) {
		return NewCommander(ctx, cfg.(*Config))
	}))
}
