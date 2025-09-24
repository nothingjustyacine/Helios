package handlers

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"time"

	"helios/models"

	"github.com/bytedance/sonic"
)

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.LoginRequest
	if err := sonic.ConfigDefault.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	// 从环境变量获取用户名和密码
	envUsername := os.Getenv("USERNAME")
	envPassword := os.Getenv("PASSWORD")

	// 验证用户名和密码
	if req.Username != envUsername || req.Password != envPassword {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// 创建认证信息结构体
	authInfo := models.AuthInfo{
		Role:     "owner",
		Username: req.Username,
		Password: req.Password,
	}

	// 序列化认证信息
	authData, err := json.Marshal(authInfo)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// 设置 cookie 过期时间（7天后）
	expires := time.Now().Add(7 * 24 * time.Hour)

	// 设置 cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "auth",
		Value:    url.QueryEscape(url.QueryEscape(string(authData))),
		Path:     "/",
		Expires:  expires,
		SameSite: http.SameSiteLaxMode,
		HttpOnly: false,
		Secure:   false,
	})

	response := map[string]interface{}{
		"ok": true,
	}

	w.Header().Set("Content-Type", "application/json")
	sonic.ConfigDefault.NewEncoder(w).Encode(response)
}
