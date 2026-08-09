package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"wind_turbine/backend/db"
	"wind_turbine/backend/router"
	"wind_turbine/backend/sim"
	"wind_turbine/backend/ws"
)

func main() {
	// 数据库文件放在 exe 同级目录
	dbPath := "wind_turbine.db"

	// 如果指定了命令行参数，用作数据库路径
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}

	// 确保目录存在
	if dir := filepath.Dir(dbPath); dir != "." && dir != "" {
		os.MkdirAll(dir, 0755)
	}

	// 初始化数据库
	if err := db.Init(dbPath); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	log.Println("数据库初始化完成:", dbPath)

	// 创建 WebSocket Hub 并启动
	hub := ws.NewHub()
	go hub.Run()
	hub.StartHeartbeat()

	// 启动数据模拟器
	simulator := sim.New(hub)
	simulator.Start()

	// 设置 Gin 模式
	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "" {
		ginMode = "debug"
	}

	// 路由
	r := router.Setup(hub)

	// 启动 HTTP 服务
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := fmt.Sprintf(":%s", port)
	log.Printf("╔════════════════════════════════════════╗")
	log.Printf("║  风电场管理系统后端                      ║")
	log.Printf("║  HTTP:  http://localhost:%s            ║", port)
	log.Printf("║  WS:    ws://localhost:%s/ws           ║", port)
	log.Printf("║  DB:    %s", dbPath)
	log.Printf("╚════════════════════════════════════════╝")

	if err := r.Run(addr); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
