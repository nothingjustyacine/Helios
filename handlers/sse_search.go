package handlers

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"helios/config"
	"helios/lib"
	"helios/models"

	"github.com/bytedance/sonic"
)

// SSESearchHandler 实现流式搜索功能
func SSESearchHandler(w http.ResponseWriter, r *http.Request) {
	// 设置 SSE 响应头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// 检查是否支持流式响应
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// 获取查询参数
	query := r.URL.Query().Get("q")
	if query == "" {
		// 返回错误响应
		w.WriteHeader(http.StatusBadRequest)
		errorResponse := map[string]string{"error": "搜索关键词不能为空"}
		errorData, _ := sonic.MarshalString(errorResponse)
		fmt.Fprintf(w, "data: %s\n\n", errorData)
		flusher.Flush()
		return
	}

	// 获取 API 站点配置
	apiSites := config.GlobalConfig.APISites
	if len(apiSites) == 0 {
		http.Error(w, "No API sites configured", http.StatusInternalServerError)
		return
	}

	// 将 map 转换为 slice
	var sites []models.APISite
	for _, site := range apiSites {
		sites = append(sites, site)
	}

	// 创建上下文用于取消操作
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// 发送开始事件
	startEvent := models.SSEStartEvent{
		Type:         "start",
		Query:        query,
		TotalSources: len(sites),
		Timestamp:    time.Now().UnixMilli(),
	}
	startData, _ := sonic.MarshalString(startEvent)
	fmt.Fprintf(w, "data: %s\n\n", startData)
	flusher.Flush()

	// 共享状态
	var (
		completedSources int
		allResults       []models.SearchResult
		mu               sync.Mutex
		streamClosed     bool
	)

	// 辅助函数：安全地发送数据
	safeSend := func(data string) bool {
		mu.Lock()
		defer mu.Unlock()

		if streamClosed {
			return false
		}

		select {
		case <-ctx.Done():
			streamClosed = true
			return false
		default:
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
			return true
		}
	}

	// 为每个源创建搜索 goroutine
	var wg sync.WaitGroup
	for _, site := range sites {
		wg.Add(1)
		go func(apiSite models.APISite) {
			defer wg.Done()

			// 创建带超时的上下文
			siteCtx, siteCancel := context.WithTimeout(ctx, 20*time.Second)
			defer siteCancel()

			// 执行搜索
			results, err := searchFromAPIWithContext(siteCtx, apiSite, query, 5)

			mu.Lock()
			completedSources++
			completedCount := completedSources
			mu.Unlock()

			if err != nil {
				// 发送错误事件
				errorEvent := models.SSESourceErrorEvent{
					Type:       "source_error",
					Source:     apiSite.Key,
					SourceName: apiSite.Name,
					Error:      err.Error(),
					Timestamp:  time.Now().UnixMilli(),
				}
				errorData, _ := sonic.MarshalString(errorEvent)
				safeSend(errorData)
				return
			}

			// 过滤黄色内容
			filteredResults := lib.FilterYellowContent(results)

			// 发送搜索结果事件
			resultEvent := models.SSESourceResultEvent{
				Type:       "source_result",
				Source:     apiSite.Key,
				SourceName: apiSite.Name,
				Results:    filteredResults,
				Timestamp:  time.Now().UnixMilli(),
			}
			resultData, _ := sonic.MarshalString(resultEvent)
			if !safeSend(resultData) {
				return
			}

			// 添加到总结果中
			mu.Lock()
			if len(filteredResults) > 0 {
				allResults = append(allResults, filteredResults...)
			}
			mu.Unlock()

			// 检查是否所有源都已完成
			mu.Lock()
			if completedCount == len(sites) {
				// 发送完成事件
				completeEvent := models.SSECompleteEvent{
					Type:             "complete",
					TotalResults:     len(allResults),
					CompletedSources: completedCount,
					Timestamp:        time.Now().UnixMilli(),
				}
				completeData, _ := sonic.MarshalString(completeEvent)
				safeSend(completeData)

				// 标记流已关闭
				streamClosed = true
			}
			mu.Unlock()
		}(site)
	}

	// 等待所有搜索完成
	wg.Wait()
}

// searchFromAPIWithContext 带上下文的 API 搜索函数
func searchFromAPIWithContext(ctx context.Context, apiSite models.APISite, query string, maxPages int) ([]models.SearchResult, error) {
	// 创建一个通道来接收搜索结果
	resultChan := make(chan []models.SearchResult, 1)
	errorChan := make(chan error, 1)

	// 在 goroutine 中执行搜索
	go func() {
		results, err := lib.SearchFromAPI(apiSite, query, maxPages)
		if err != nil {
			errorChan <- err
			return
		}

		resultChan <- results
	}()

	// 等待结果或上下文取消
	select {
	case results := <-resultChan:
		return results, nil
	case err := <-errorChan:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
