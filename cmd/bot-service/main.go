package main

import (
	"fmt"

	// 飞书 SDK
	_ "github.com/larksuite/oapi-sdk-go/v3"

	// CLI 框架
	_ "github.com/spf13/cobra"

	// 配置管理
	_ "github.com/spf13/viper"

	// 日志
	_ "go.uber.org/zap"

	// YAML 解析
	_ "gopkg.in/yaml.v3"
)

func main() {
	fmt.Println("飞书机器人服务启动中...")
}
