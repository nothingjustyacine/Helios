package handlers

import (
	"net/http"
	"regexp"
	"strconv"

	"helios/config"
	"helios/lib"

	"github.com/bytedance/sonic"
)

func DetailHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取查询参数
	id := r.URL.Query().Get("id")
	sourceCode := r.URL.Query().Get("source")

	// 验证必要参数
	if id == "" || sourceCode == "" {
		http.Error(w, "缺少必要参数", http.StatusBadRequest)
		return
	}

	// 验证 ID 格式
	if !regexp.MustCompile(`^[\w-]+$`).MatchString(id) {
		http.Error(w, "无效的视频ID格式", http.StatusBadRequest)
		return
	}

	// 获取 API 站点配置
	apiSite, exists := config.GlobalConfig.APISites[sourceCode]
	if !exists {
		http.Error(w, "无效的API来源", http.StatusBadRequest)
		return
	}

	// 调用详情获取函数
	result, err := lib.GetDetailFromAPI(apiSite, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 设置缓存头
	cacheTime := 7200 // 2小时
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age="+strconv.Itoa(cacheTime)+", s-maxage="+strconv.Itoa(cacheTime))
	w.Header().Set("CDN-Cache-Control", "public, s-maxage="+strconv.Itoa(cacheTime))
	w.Header().Set("Vercel-CDN-Cache-Control", "public, s-maxage="+strconv.Itoa(cacheTime))
	w.Header().Set("Netlify-Vary", "query")

	// 返回结果
	sonic.ConfigDefault.NewEncoder(w).Encode(result)
}
