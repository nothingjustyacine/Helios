package lib

import (
	"fmt"
	"sync"
	"time"

	"helios/models"
)

// 缓存配置常量
const (
	// 搜索缓存 TTL：10分钟
	SearchCacheTTL = 10 * time.Minute
	// 缓存清理间隔：5分钟
	CacheCleanupInterval = 5 * time.Minute
	// 最大缓存条目数量
	MaxCacheSize = 1000
)

// CacheManager 缓存管理器
type CacheManager struct {
	cache        map[string]CachedSearchPage
	mutex        sync.RWMutex
	cleanupTimer *time.Timer
	lastCleanup  time.Time
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

	// 检查缓存是否过期
	if time.Now().After(cached.ExpiresAt) {
		// 异步删除过期缓存
		go cm.Delete(key)
		return nil, false
	}

	return &cached, true
}

// Set 设置缓存
func (cm *CacheManager) Set(key string, data CachedSearchPage) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// 确保自动清理已启动
	cm.ensureAutoCleanupStarted()

	// 惰性清理：每次写入时检查是否需要清理
	now := time.Now()
	if now.Sub(cm.lastCleanup) > CacheCleanupInterval {
		cm.performCacheCleanup()
	}

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

	cm.performCacheCleanup()
}

// ensureAutoCleanupStarted 确保自动清理已启动（惰性初始化）
func (cm *CacheManager) ensureAutoCleanupStarted() {
	if cm.cleanupTimer == nil {
		cm.startAutoCleanup()
	}
}

// performCacheCleanup 智能清理过期的缓存条目
func (cm *CacheManager) performCacheCleanup() {
	now := time.Now()
	var keysToDelete []string
	sizeLimitedDeleted := 0

	// 1. 清理过期条目
	for key, entry := range cm.cache {
		if now.After(entry.ExpiresAt) {
			keysToDelete = append(keysToDelete, key)
		}
	}

	expiredCount := len(keysToDelete)
	for _, key := range keysToDelete {
		delete(cm.cache, key)
	}

	// 2. 如果缓存大小超限，清理最老的条目（LRU策略）
	if len(cm.cache) > MaxCacheSize {
		// 将缓存条目转换为切片进行排序
		type cacheEntry struct {
			key       string
			expiresAt time.Time
		}

		var entries []cacheEntry
		for key, cached := range cm.cache {
			entries = append(entries, cacheEntry{
				key:       key,
				expiresAt: cached.ExpiresAt,
			})
		}

		// 按照过期时间排序，最早过期的在前面
		for i := 0; i < len(entries)-1; i++ {
			for j := i + 1; j < len(entries); j++ {
				if entries[i].expiresAt.After(entries[j].expiresAt) {
					entries[i], entries[j] = entries[j], entries[i]
				}
			}
		}

		toRemove := len(cm.cache) - MaxCacheSize
		for i := 0; i < toRemove && i < len(entries); i++ {
			delete(cm.cache, entries[i].key)
			sizeLimitedDeleted++
		}
	}

	cm.lastCleanup = now

	// 输出清理统计信息（可选）
	if expiredCount > 0 || sizeLimitedDeleted > 0 {
		fmt.Printf("缓存清理完成: 过期=%d, 大小限制=%d, 剩余=%d\n",
			expiredCount, sizeLimitedDeleted, len(cm.cache))
	}
}

// startAutoCleanup 启动自动清理定时器
func (cm *CacheManager) startAutoCleanup() {
	if cm.cleanupTimer != nil {
		return // 避免重复启动
	}

	cm.cleanupTimer = time.AfterFunc(CacheCleanupInterval, func() {
		cm.CleanupExpired()
		// 重新设置定时器
		cm.cleanupTimer = time.AfterFunc(CacheCleanupInterval, func() {
			cm.CleanupExpired()
		})
	})
}

// GetCacheKey 生成缓存键
func GetCacheKey(apiSiteKey, query string, page int) string {
	return fmt.Sprintf("%s:%s:%d", apiSiteKey, query, page)
}

// 全局缓存管理器实例
var globalCacheManager = NewCacheManager()

// GetCachedSearchPage 获取缓存的搜索页面（全局函数）
func GetCachedSearchPage(apiSiteKey, query string, page int) *CachedSearchPage {
	key := GetCacheKey(apiSiteKey, query, page)
	cached, exists := globalCacheManager.Get(key)
	if !exists {
		return nil
	}
	return cached
}

// SetCachedSearchPage 设置缓存的搜索页面（全局函数）
func SetCachedSearchPage(apiSiteKey, query string, page int, status string, data []models.SearchResult, pageCount *int) {
	key := GetCacheKey(apiSiteKey, query, page)
	now := time.Now()
	cached := CachedSearchPage{
		Status:    status,
		Data:      data,
		PageCount: pageCount,
		Timestamp: now,
		ExpiresAt: now.Add(SearchCacheTTL),
	}
	globalCacheManager.Set(key, cached)
}
