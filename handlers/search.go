package handlers

import (
	"context"
	"fmt"
	"helios/config"
	"net/http"
	"sync"
	"time"

	"helios/lib"
	"helios/models"

	"github.com/bytedance/sonic"
)

// AuthInfo 认证信息结构
type AuthInfo struct {
	Role     string `json:"role"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func SearchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query().Get("q")

	// 如果没有查询参数，返回空结果并设置缓存头
	if query == "" {
		cacheTime := 7200
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, s-maxage=%d", cacheTime, cacheTime))
		w.Header().Set("CDN-Cache-Control", fmt.Sprintf("public, s-maxage=%d", cacheTime))
		w.Header().Set("Vercel-CDN-Cache-Control", fmt.Sprintf("public, s-maxage=%d", cacheTime))
		w.Header().Set("Netlify-Vary", "query")

		response := map[string]interface{}{
			"results": []interface{}{},
		}
		sonic.ConfigDefault.NewEncoder(w).Encode(response)
		return
	}

	// 获取配置和可用的API站点
	apiSites := config.GlobalConfig.APISites

	// 创建带超时控制的搜索任务
	var wg sync.WaitGroup
	var mutex sync.Mutex
	var allResults []models.SearchResult

	// 为每个API站点创建搜索任务
	for _, site := range apiSites {
		wg.Add(1)
		go func(apiSite models.APISite) {
			defer wg.Done()

			// 创建带超时的上下文
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			// 使用channel来接收结果或超时
			resultChan := make(chan []models.SearchResult, 1)

			go func() {
				results, _ := lib.SearchFromAPI(apiSite, query, 5)
				resultChan <- results
			}()

			select {
			case results := <-resultChan:
				if len(results) > 0 {
					mutex.Lock()
					allResults = append(allResults, results...)
					mutex.Unlock()
				}
			case <-ctx.Done():
				fmt.Printf("搜索超时 %s\n", apiSite.Name)
			}
		}(site)
	}

	// 等待所有搜索任务完成
	wg.Wait()

	// 应用黄色过滤
	allResults = lib.FilterYellowContent(allResults)

	// 设置缓存头
	cacheTime := 7200

	// 如果没有结果，返回空结果且不缓存
	if len(allResults) == 0 {
		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"results": []interface{}{},
		}
		sonic.ConfigDefault.NewEncoder(w).Encode(response)
		return
	}

	// 返回结果并设置缓存头
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, s-maxage=%d", cacheTime, cacheTime))
	w.Header().Set("CDN-Cache-Control", fmt.Sprintf("public, s-maxage=%d", cacheTime))
	w.Header().Set("Vercel-CDN-Cache-Control", fmt.Sprintf("public, s-maxage=%d", cacheTime))
	w.Header().Set("Netlify-Vary", "query")

	response := map[string]interface{}{
		"results": allResults,
	}
	sonic.ConfigDefault.NewEncoder(w).Encode(response)
}
