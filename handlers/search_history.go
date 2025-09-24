package handlers

import (
	"net/http"

	"helios/database"
	"helios/models"

	"github.com/bytedance/sonic"
)

func SearchHistoryHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleGetSearchHistory(w, r)
	case http.MethodPost:
		handlePostSearchHistory(w, r)
	case http.MethodDelete:
		handleDeleteSearchHistory(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleGetSearchHistory(w http.ResponseWriter, _ *http.Request) {
	// 获取搜索历史记录
	record, err := database.GetSearchHistory()
	if err != nil {
		http.Error(w, "Failed to get search history", http.StatusInternalServerError)
		return
	}

	var keywords []string
	if record != "" {
		// 解析JSON数组
		if err := sonic.UnmarshalString(record, &keywords); err != nil {
			http.Error(w, "Failed to parse search history", http.StatusInternalServerError)
			return
		}
	}

	// 直接返回关键词数组
	w.Header().Set("Content-Type", "application/json")
	sonic.ConfigDefault.NewEncoder(w).Encode(keywords)
}

func handlePostSearchHistory(w http.ResponseWriter, r *http.Request) {
	var req models.SearchHistoryRequest
	if err := sonic.ConfigDefault.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Keyword == "" {
		http.Error(w, "keyword parameter is required", http.StatusBadRequest)
		return
	}

	// 在事务中执行获取和更新操作
	keywords, err := database.UpdateSearchHistoryInTransaction(req.Keyword)
	if err != nil {
		http.Error(w, "Failed to update search history", http.StatusInternalServerError)
		return
	}

	// 直接返回关键词数组
	w.Header().Set("Content-Type", "application/json")
	sonic.ConfigDefault.NewEncoder(w).Encode(keywords)
}

func handleDeleteSearchHistory(w http.ResponseWriter, r *http.Request) {
	keyword := r.URL.Query().Get("keyword")
	if keyword == "" {
		http.Error(w, "keyword parameter is required", http.StatusBadRequest)
		return
	}

	// 在事务中删除关键词并返回更新后的数组
	if database.DeleteSearchHistoryKeyword(keyword) != nil {
		http.Error(w, "Failed to delete search history keyword", http.StatusInternalServerError)
		return
	}

	// 直接返回更新后的关键词数组
	response := map[string]interface{}{
		"success": true,
	}

	w.Header().Set("Content-Type", "application/json")
	sonic.ConfigDefault.NewEncoder(w).Encode(response)
}
