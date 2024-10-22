package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/danger-dream/ebpf-firewall/internal/config"
	"github.com/danger-dream/ebpf-firewall/internal/ebpf"
	"github.com/danger-dream/ebpf-firewall/internal/interfaces"
	"github.com/danger-dream/ebpf-firewall/internal/processor"
	"github.com/danger-dream/ebpf-firewall/internal/websocket"

	"github.com/oschwald/geoip2-golang"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("运行失败: %v", err)
	}
}

func run() error {
	configPath := parseFlags()
	configManager, err := config.NewConfigManager(configPath)
	if err != nil {
		return fmt.Errorf("加载配置失败: %v", err)
	}

	app, err := setupApplication(configManager)
	if err != nil {
		return fmt.Errorf("设置应用程序失败: %v", err)
	}

	if err := app.Start(); err != nil {
		return fmt.Errorf("启动应用程序失败: %v", err)
	}

	waitForShutdown(app)

	return nil
}

func parseFlags() string {
	var configPath string
	flag.StringVar(&configPath, "c", "", "配置文件路径")
	flag.Parse()

	if configPath == "" {
		if _, err := os.Stat("config.yaml"); err == nil {
			configPath = "config.yaml"
		} else if _, err := os.Stat("config.json"); err == nil {
			configPath = "config.json"
		} else {
			panic("运行目录下不存在 config.yaml 或 config.json 文件，请指定配置文件路径")
		}
	}
	return configPath
}

func setupApplication(configManager *config.ConfigManager) (*Application, error) {
	var geoipDB *geoip2.Reader
	if configManager.Config.GeoIPPath != "" {
		var err error
		geoipDB, err = geoip2.Open(configManager.Config.GeoIPPath)
		if err != nil {
			return nil, fmt.Errorf("打开 GeoIP 数据库失败: %v", err)
		}
	}

	ruleMatcher := processor.NewRuleMatcher(configManager.Config.Rules)
	wsServer := websocket.NewWebSocketServer(getFileSystem())

	summaryManager := ebpf.NewSummary(configManager, ruleMatcher, wsServer, geoipDB)
	ebpfManager := ebpf.NewEBPFManager(configManager, summaryManager)

	wsMessageHandler := websocket.NewWebSocketMessageHandler(
		configManager,
		ebpfManager,
		wsServer,
		ruleMatcher,
		summaryManager,
	)
	wsServer.SetMessageHandler(wsMessageHandler)

	return &Application{
		ebpfManager:   ebpfManager,
		wsServer:      wsServer,
		configManager: configManager,
	}, nil
}

type Application struct {
	ebpfManager   *ebpf.EBPFManager
	wsServer      interfaces.WebSocketServer
	configManager *config.ConfigManager
}

func (app *Application) Start() error {
	if err := app.ebpfManager.Start(); err != nil {
		return fmt.Errorf("启动 eBPF 管理器失败: %v", err)
	}
	go app.wsServer.Start(app.configManager.Config.Port)
	return nil
}

func (app *Application) Shutdown() {
	log.Println("正在关闭应用程序...")
	// 关闭 eBPF 管理器
	app.ebpfManager.Shutdown()
	// 关闭 WebSocket 服务器
	if err := app.wsServer.Shutdown(); err != nil {
		log.Printf("关闭 WebSocket 服务器时发生错误: %v", err)
	}
	log.Println("应用程序已关闭")
}

func waitForShutdown(app *Application) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
	app.Shutdown()
}
