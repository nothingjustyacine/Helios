package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"helios/database"
	"helios/models"

	"github.com/bytedance/sonic"
)

func PlayRecordsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleGetPlayRecords(w, r)
	case http.MethodPost:
		handlePostPlayRecords(w, r)
	case http.MethodDelete:
		handleDeletePlayRecords(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleGetPlayRecords(w http.ResponseWriter, r *http.Request) {
	records, err := database.GetAllPlayRecords()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get play records: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	sonic.ConfigDefault.NewEncoder(w).Encode(records)
}

func handlePostPlayRecords(w http.ResponseWriter, r *http.Request) {
	var req models.PlayRecordPostRequest
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	if err := sonic.UnmarshalString(string(body), &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 从 key 中提取 source 和 source_id
	// key 格式为 "source+source_id"，例如 "dbzy+97838"
	parts := strings.Split(req.Key, "+")
	if len(parts) != 2 {
		http.Error(w, "Invalid key format. Expected format: source+source_id", http.StatusBadRequest)
		return
	}

	source := parts[0]
	sourceID := parts[1]

	// 设置 record 的 source 和 source_id
	req.Record.Source = source
	req.Record.SourceID = sourceID

	// 使用 UPSERT 操作插入或更新记录，避免竞态条件
	err = database.UpsertPlayRecord(&req.Record)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to upsert play record: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
	}

	w.Header().Set("Content-Type", "application/json")
	sonic.ConfigDefault.NewEncoder(w).Encode(response)
}

func handleDeletePlayRecords(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")

	// 如果 key 为空，删除全部播放记录
	if key == "" {
		err := database.DeleteAllPlayRecords()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to delete all play records: %v", err), http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"success": true,
		}

		w.Header().Set("Content-Type", "application/json")
		sonic.ConfigDefault.NewEncoder(w).Encode(response)
		return
	}

	// 从 key 中提取 source 和 source_id
	// key 格式为 "source+source_id"，例如 "dbzy+101851"
	parts := strings.Split(key, "+")
	if len(parts) != 2 {
		http.Error(w, "Invalid key format. Expected format: source+source_id", http.StatusBadRequest)
		return
	}

	source := parts[0]
	sourceID := parts[1]

	// 删除记录（无论记录是否存在都返回成功）
	err := database.DeletePlayRecord(source, sourceID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete play record: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
	}

	w.Header().Set("Content-Type", "application/json")
	sonic.ConfigDefault.NewEncoder(w).Encode(response)
}
