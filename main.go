package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/seb/tractor-vin/api"
	"github.com/seb/tractor-vin/decoder"
)

func main() {
	decodeCmd := flag.NewFlagSet("decode", flag.ExitOnError)
	serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)
	brandsCmd := flag.NewFlagSet("brands", flag.ExitOnError)
	port := serveCmd.Int("port", 8080, "API server port")

	if len(os.Args) < 2 {
		fmt.Println("TractorVIN — 拖拉机序列号解码器")
		fmt.Println()
		fmt.Println("命令:")
		fmt.Println("  decode <序列号>    解码拖拉机序列号")
		fmt.Println("  serve              启动 Web API 服务")
		fmt.Println("  brands             列出支持的品牌")
		os.Exit(0)
	}

	switch os.Args[1] {
	case "decode":
		decodeCmd.Parse(os.Args[2:])
		args := decodeCmd.Args()
		if len(args) < 1 {
			fmt.Println("用法: tractor-vin decode <序列号>")
			os.Exit(1)
		}
		result, err := decoder.Decode(args[0])
		if err != nil {
			fmt.Printf("❌ 解码失败: %v\n", err)
			os.Exit(1)
		}
		printResult(result)

	case "serve":
		serveCmd.Parse(os.Args[2:])
		fmt.Printf("🚜 TractorVIN API 启动在 http://localhost:%d\n", *port)
		fmt.Println("接口: POST /decode | GET /brands | GET /health")
		api.Serve(*port)

	case "brands":
		brandsCmd.Parse(os.Args[2:])
		fmt.Println("支持的品牌:")
		fmt.Println()
		for _, b := range decoder.Brands() {
			fmt.Printf("  • %s\n", b)
		}

	default:
		fmt.Printf("未知命令: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func printResult(d *decoder.DecodedInfo) {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║        🚜 拖拉机序列号解码结果        ║")
	fmt.Println("╠══════════════════════════════════════╣")
	fmt.Printf("║ 品牌:       %-25s ║\n", d.Brand)
	fmt.Printf("║ 型号:       %-25s ║\n", d.Model)
	fmt.Printf("║ 系列:       %-25s ║\n", d.Series)
	fmt.Printf("║ 生产年份:   %-25s ║\n", d.Year)
	fmt.Printf("║ 生产工厂:   %-25s ║\n", d.Factory)
	fmt.Printf("║ 发动机系列: %-25s ║\n", d.EngineFamily)
	fmt.Printf("║ 生产序号:   %-25s ║\n", d.ProductionNumber)
	if d.Country != "" {
		fmt.Printf("║ 生产国家:   %-25s ║\n", d.Country)
	}
	fmt.Println("╚══════════════════════════════════════╝")
	fmt.Println()
}
