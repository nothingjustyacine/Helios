package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"helios/handlers"
	"helios/scheduler"
)

func validateEnvironment() {
	username := os.Getenv("USERNAME")
	password := os.Getenv("PASSWORD")
	subscriptionURL := os.Getenv("SUBSCRIPTION_URL")

	if username == "" {
		log.Fatal("Error: USERNAME environment variable is not set")
	}
	if username == "your_username" {
		log.Fatal("Error: Please change your USERNAME environment variable")
	}
	if password == "" {
		log.Fatal("Error: PASSWORD environment variable is not set")
	}
	if password == "your_password" {
		log.Fatal("Error: Please change your PASSWORD environment variable")
	}
	if subscriptionURL == "" {
		log.Fatal("Error: SUBSCRIPTION_URL environment variable is not set")
	}
	if subscriptionURL == "https://your_subscription_url.com" {
		log.Fatal("Error: Please change your SUBSCRIPTION_URL environment variable")
	}
}

func main() {
	validateEnvironment()

	initAll()

	// 启动定时任务调度器
	go startScheduler()

	// 登录接口不需要认证
	http.HandleFunc("/api/login", handlers.LoginHandler)

	// 其他接口都需要认证中间件
	http.HandleFunc("/api/search", handlers.AuthMiddleware(handlers.SearchHandler))
	http.HandleFunc("/api/search/ws", handlers.AuthMiddleware(handlers.SSESearchHandler))
	http.HandleFunc("/api/detail", handlers.AuthMiddleware(handlers.DetailHandler))
	http.HandleFunc("/api/favorites", handlers.AuthMiddleware(handlers.FavoritesHandler))
	http.HandleFunc("/api/searchhistory", handlers.AuthMiddleware(handlers.SearchHistoryHandler))
	http.HandleFunc("/api/playrecords", handlers.AuthMiddleware(handlers.PlayRecordsHandler))

	port := ":8080"
	fmt.Printf("Server starting on port %s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

// startScheduler 启动定时任务调度器
func startScheduler() {
	sched := scheduler.NewScheduler()

	// 添加每小时执行的任务
	hourlyTask := scheduler.NewHourlyTask("每小时数据清理任务")
	sched.AddTask(hourlyTask)

	// 启动调度器
	sched.Start()
}
