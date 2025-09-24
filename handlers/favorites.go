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

func FavoritesHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleGetFavorites(w, r)
	case http.MethodPost:
		handlePostFavorites(w, r)
	case http.MethodDelete:
		handleDeleteFavorites(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleGetFavorites(w http.ResponseWriter, r *http.Request) {
	// 从数据库获取所有收藏夹记录
	favorites, err := database.GetAllFavorites()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get favorites: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	sonic.ConfigDefault.NewEncoder(w).Encode(favorites)
}

func handlePostFavorites(w http.ResponseWriter, r *http.Request) {
	var req models.FavoritePostRequest
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

	// 设置 favorite 的 source 和 source_id
	req.Favorite.Source = source
	req.Favorite.SourceID = sourceID

	// 使用 UPSERT 操作插入或更新记录，避免竞态条件
	err = database.UpsertFavorite(&req.Favorite)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to upsert favorite: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
	}

	w.Header().Set("Content-Type", "application/json")
	sonic.ConfigDefault.NewEncoder(w).Encode(response)
}

func handleDeleteFavorites(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "key parameter is required", http.StatusBadRequest)
		return
	}

	// 从 key 中提取 source 和 source_id
	// key 格式为 "source+source_id"，例如 "dbzy+97838"
	parts := strings.Split(key, "+")
	if len(parts) != 2 {
		http.Error(w, "Invalid key format. Expected format: source+source_id", http.StatusBadRequest)
		return
	}

	source := parts[0]
	sourceID := parts[1]

	// 删除记录（无论记录是否存在都返回成功）
	err := database.DeleteFavorite(source, sourceID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete favorite: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
	}

	w.Header().Set("Content-Type", "application/json")
	sonic.ConfigDefault.NewEncoder(w).Encode(response)
}
