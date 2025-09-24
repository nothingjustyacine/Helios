package handlers

import (
	"bytes"
	"net/http"
	"net/url"
	"os"

	"helios/models"

	"github.com/bytedance/sonic"
)

// AuthMiddleware 认证中间件，校验cookie中的用户名密码
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 获取cookie
		cookie, err := r.Cookie("auth")
		if err != nil {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		// URL解码cookie值
		decodedValue, err := url.QueryUnescape(cookie.Value)
		if err != nil {
			http.Error(w, "Invalid authentication data", http.StatusUnauthorized)
			return
		}
		decodedValue, err = url.QueryUnescape(decodedValue)
		if err != nil {
			http.Error(w, "Invalid authentication data", http.StatusUnauthorized)
			return
		}

		// 解析cookie中的认证信息
		var authInfo models.AuthInfo
		if err := sonic.ConfigDefault.NewDecoder(bytes.NewReader([]byte(decodedValue))).Decode(&authInfo); err != nil {
			http.Error(w, "Invalid authentication data", http.StatusUnauthorized)
			return
		}

		// 从环境变量获取用户名和密码
		envUsername := os.Getenv("USERNAME")
		envPassword := os.Getenv("PASSWORD")

		// 验证用户名和密码
		if authInfo.Username != envUsername || authInfo.Password != envPassword {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		// 认证通过，继续处理请求
		next.ServeHTTP(w, r)
	}
}
