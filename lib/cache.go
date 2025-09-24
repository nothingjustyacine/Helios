package lib

import (
	"fmt"
	"sync"
	"time"

	"helios/models"
)

// CacheManager 缓存管理器
type CacheManager struct {
	cache map[string]CachedSearchPage
	mutex sync.RWMutex
}

// NewCacheManager 创建新的缓存管理器
func NewCacheManager() *CacheManager {
	return &CacheManager{
		cache: make(map[string]CachedSearchPage),
	}
}

// Get 获取缓存
func (cm *CacheManager) Get(key string) (*CachedSearchPage, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	cached, exists := cm.cache[key]
	if !exists {
		return nil, false
	}

	// 检查缓存是否过期（5分钟）
	if time.Since(cached.Timestamp) > 5*time.Minute {
		return nil, false
	}

	return &cached, true
}

// Set 设置缓存
func (cm *CacheManager) Set(key string, data CachedSearchPage) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.cache[key] = data
}

// Delete 删除缓存
func (cm *CacheManager) Delete(key string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	delete(cm.cache, key)
}

// Clear 清空缓存
func (cm *CacheManager) Clear() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.cache = make(map[string]CachedSearchPage)
}

// Size 获取缓存大小
func (cm *CacheManager) Size() int {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	return len(cm.cache)
}

// CleanupExpired 清理过期缓存
func (cm *CacheManager) CleanupExpired() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	now := time.Now()
	for key, cached := range cm.cache {
		if now.Sub(cached.Timestamp) > 5*time.Minute {
			delete(cm.cache, key)
		}
	}
}

// GetCacheKey 生成缓存键
func GetCacheKey(apiSiteKey, query string, page int) string {
	return fmt.Sprintf("%s:%s:%d", apiSiteKey, query, page)
}

// 全局缓存管理器实例
var globalCacheManager = NewCacheManager()

// GetCachedSearchPage 获取缓存的搜索页面（全局函数）
func GetCachedSearchPage(apiSiteKey, query string, page int) *CachedSearchPage {
	fmt.Println("GetCachedSearchPage", apiSiteKey, query, page)
	key := GetCacheKey(apiSiteKey, query, page)
	cached, exists := globalCacheManager.Get(key)
	if !exists {
		return nil
	}
	return cached
}

// SetCachedSearchPage 设置缓存的搜索页面（全局函数）
func SetCachedSearchPage(apiSiteKey, query string, page int, status string, data []models.SearchResult, pageCount *int) {
	fmt.Println("SetCachedSearchPage", apiSiteKey, query, page, status, pageCount)
	key := GetCacheKey(apiSiteKey, query, page)
	cached := CachedSearchPage{
		Status:    status,
		Data:      data,
		PageCount: pageCount,
		Timestamp: time.Now(),
	}
	globalCacheManager.Set(key, cached)
}
