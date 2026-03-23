package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	bots    string
)

var rootCmd = &cobra.Command{
	Use:   "bot-service",
	Short: "飞书机器人服务",
	Long:  `多机器人协作服务，支持配置文件管理和任务分发`,
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "启动服务",
	Run:   runStart,
}

func init() {
	cobra.OnInitialize(initConfig)

	// 全局标志
	startCmd.Flags().StringVarP(&cfgFile, "config", "c", "configs/bots.yaml", "配置文件路径")
	startCmd.Flags().StringVarP(&bots, "bots", "b", "", "要启动的机器人列表（逗号分隔），不指定则启动所有")

	// 添加 --all 标志
	startCmd.Flags().Bool("all", false, "启动所有机器人")

	rootCmd.AddCommand(startCmd)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath("configs")
		viper.SetConfigName("bots")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("读取配置文件失败: %v\n", err)
		os.Exit(1)
	}
}

func runStart(cmd *cobra.Command, args []string) {
	// 获取配置文件路径
	configPath, _ := cmd.Flags().GetString("config")

	// 加载配置
	fmt.Printf("正在加载配置: %s\n", configPath)
	// TODO: 实现配置加载

	// 获取要启动的机器人
	botList, _ := cmd.Flags().GetString("bots")
	startAll, _ := cmd.Flags().GetBool("all")

	if startAll {
		fmt.Println("启动所有机器人...")
	} else if botList != "" {
		fmt.Printf("启动机器人: %s\n", botList)
	} else {
		fmt.Println("未指定机器人，启动所有...")
	}

	// TODO: 实现服务启动逻辑

	fmt.Println("服务启动中...")
	select {} // 阻塞主线程
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
