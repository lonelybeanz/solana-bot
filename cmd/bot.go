/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"log"
	"solana-bot/internal/config"
	"solana-bot/internal/monitor"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
)

// botCmd represents the bot command
var botCmd = &cobra.Command{
	Use:   "bot",
	Short: "solana-bot bot",
	Long:  `solana-bot bot`,
	Run: func(cmd *cobra.Command, args []string) {
		StartPumpMonitor(cmd)
		Start(cfgFile)
	},
}

func init() {
	rootCmd.AddCommand(botCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// botCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// botCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	// 添加本地 flags
	botCmd.Flags().Bool("mint", false, "启用 mint 监控")
	botCmd.Flags().Bool("smart", false, "启用 smart 监控")
	botCmd.Flags().Bool("scm", false, "启用 scm 监控")

}

func StartPumpMonitor(botCmd *cobra.Command) {

	godotenv.Load()

	var c config.Config
	conf.MustLoad(cfgFile, &c)
	config.C = c

	// 获取命令行参数
	mintEnable, _ := botCmd.Flags().GetBool("mint")
	smartEnable, _ := botCmd.Flags().GetBool("smart")
	scmEnable, _ := botCmd.Flags().GetBool("scm")

	// 设置 MonitorType
	monitor.SetMonitorType(map[string]bool{
		"mint":  mintEnable,
		"smart": smartEnable,
		"scm":   scmEnable,
	})

	monitor.InitConfig()

	// 启动配置监听
	restartChan := make(chan struct{}, 1)
	go monitor.WatchConfigChanges(restartChan)

	//
	// robotM, err := monitor.NewRobotMonitor()
	// if err != nil {
	// 	logx.Error(err)
	// 	return
	// }
	// go robotM.Start()

	pumpMonitor, err := monitor.NewPumpFunMonitor()
	if err != nil {
		logx.Error(err)
		return
	}
	monitor.PumpMonitor = pumpMonitor
	pumpMonitor.Start()
	// 监听重启信号
	go func() {
		for range restartChan {
			log.Println("准备重启服务...")
			pumpMonitor.Stop()
			newM, err := monitor.NewPumpFunMonitor()
			if err != nil {
				logx.Must(err)
				return
			}
			monitor.PumpMonitor = newM
			newM.Start()
		}
	}()

	// ln, err := net.Listen("tcp", ":9999")
	// if err != nil {
	// 	log.Fatalf("监听失败: %v", err)
	// }
	// defer ln.Close()

	// fmt.Printf("服务运行中，监听端口: %s\n", ln.Addr().String())

	// // 阻塞接受连接（不处理，纯粹阻塞）
	// for {
	// 	conn, err := ln.Accept()
	// 	if err != nil {
	// 		log.Printf("接受连接失败: %v", err)
	// 		continue
	// 	}
	// 	conn.Close() // 立即关闭连接，只为阻塞服务
	// }

}
