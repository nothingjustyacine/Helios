package scheduler

import (
	"log"
	"os"
	"sync"
	"time"

	"helios/config"
	"helios/database"
	"helios/lib"
	"helios/models"
)

// ScheduledTask 定时任务接口
type ScheduledTask interface {
	Execute() error
	GetName() string
}

// Scheduler 定时任务调度器
type Scheduler struct {
	tasks []ScheduledTask
}

// NewScheduler 创建新的调度器
func NewScheduler() *Scheduler {
	return &Scheduler{
		tasks: make([]ScheduledTask, 0),
	}
}

// AddTask 添加定时任务
func (s *Scheduler) AddTask(task ScheduledTask) {
	s.tasks = append(s.tasks, task)
	log.Printf("已添加定时任务: %s", task.GetName())
}

// Start 启动调度器
func (s *Scheduler) Start() {
	log.Println("定时任务调度器启动中...")

	// 每小时执行一次
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	// 立即执行一次
	s.executeAllTasks()

	// 定时执行
	for range ticker.C {
		s.executeAllTasks()
	}
}

// executeAllTasks 执行所有任务
func (s *Scheduler) executeAllTasks() {
	log.Println("开始执行定时任务...")

	for _, task := range s.tasks {
		go func(t ScheduledTask) {
			log.Printf("执行任务: %s", t.GetName())
			if err := t.Execute(); err != nil {
				log.Printf("任务 %s 执行失败: %v", t.GetName(), err)
			} else {
				log.Printf("任务 %s 执行成功", t.GetName())
			}
		}(task)
	}
}

// HourlyTask 示例定时任务
type HourlyTask struct {
	name string
}

// NewHourlyTask 创建新的每小时任务
func NewHourlyTask(name string) *HourlyTask {
	return &HourlyTask{
		name: name,
	}
}

// Execute 执行任务
func (h *HourlyTask) Execute() error {
	go func() {
		refreshSubscription()
		refreshRecords()
	}()

	return nil
}

func refreshSubscription() {
	subscriptionURL := os.Getenv("SUBSCRIPTION_URL")
	if subscriptionURL == "" {
		log.Printf("Warning: SUBSCRIPTION_URL environment variable is not set, skipping subscription refresh")
		return
	}

	log.Println("Refreshing subscription configuration...")
	if err := config.FetchSubscription(subscriptionURL); err != nil {
		log.Printf("Failed to refresh subscription: %v", err)
		return
	}

	log.Println("Subscription configuration refreshed successfully")
}

// VideoDetailResult 视频详情获取结果
type VideoDetailResult struct {
	Key    string
	Detail *models.SearchResult
	Error  error
}

func refreshRecords() {
	log.Println("开始刷新播放记录和收藏夹记录...")

	// 获取所有播放记录
	playRecords, err := database.GetAllPlayRecords()
	if err != nil {
		log.Printf("获取播放记录失败: %v", err)
		return
	}

	// 获取所有收藏夹记录
	favorites, err := database.GetAllFavorites()
	if err != nil {
		log.Printf("获取收藏夹记录失败: %v", err)
		return
	}

	// 收集所有需要获取详情的 source+source_id 组合
	uniqueKeys := make(map[string]bool)

	// 从播放记录中收集
	for key := range playRecords {
		uniqueKeys[key] = true
	}

	// 从收藏夹记录中收集
	for key := range favorites {
		uniqueKeys[key] = true
	}

	log.Printf("需要获取详情的视频数量: %d", len(uniqueKeys))

	// 批量获取所有视频详情
	detailResults := batchFetchVideoDetails(uniqueKeys, playRecords, favorites)

	// 更新播放记录
	updatePlayRecords(playRecords, detailResults)

	// 更新收藏夹记录
	updateFavorites(favorites, detailResults)

	log.Println("播放记录和收藏夹记录刷新完成")
}

// batchFetchVideoDetails 批量获取视频详情
func batchFetchVideoDetails(uniqueKeys map[string]bool, playRecords map[string]models.PlayRecord, favorites map[string]models.Favorite) map[string]VideoDetailResult {
	results := make(map[string]VideoDetailResult)
	var wg sync.WaitGroup
	var mutex sync.Mutex

	// 限制并发数量，避免过多请求
	semaphore := make(chan struct{}, 10)

	for key := range uniqueKeys {
		wg.Add(1)
		go func(k string) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 确定使用哪个记录作为参考（优先使用播放记录）
			var source, sourceID, title string
			if record, exists := playRecords[k]; exists {
				source = record.Source
				sourceID = record.SourceID
				title = record.Title
			} else if favorite, exists := favorites[k]; exists {
				source = favorite.Source
				sourceID = favorite.SourceID
				title = favorite.Title
			}

			// 调用 FetchVideoDetail 获取最新详情
			detail, err := lib.FetchVideoDetail(lib.FetchVideoDetailOptions{
				Source:        source,
				ID:            sourceID,
				FallbackTitle: title,
			})

			mutex.Lock()
			results[k] = VideoDetailResult{Key: k, Detail: detail, Error: err}
			mutex.Unlock()
		}(key)
	}

	wg.Wait()
	return results
}

// updatePlayRecords 更新播放记录
func updatePlayRecords(playRecords map[string]models.PlayRecord, detailResults map[string]VideoDetailResult) {
	log.Println("开始更新播放记录...")
	updatedCount := 0

	for key, record := range playRecords {
		result, exists := detailResults[key]
		if !exists {
			log.Printf("未找到播放记录 [%s] 的详情结果", key)
			continue
		}

		if result.Error != nil {
			log.Printf("获取视频详情失败 [%s]: %v", key, result.Error)
			continue
		}

		if result.Detail == nil {
			log.Printf("视频详情为空 [%s]", key)
			continue
		}

		// 检查 total_episodes 是否不同
		if len(result.Detail.Episodes) != record.TotalEpisodes {
			log.Printf("发现集数变化 [%s]: %d -> %d", key, record.TotalEpisodes, len(result.Detail.Episodes))

			// 更新播放记录（只更新 title、cover、year、total_episodes）
			updatedRecord := record
			updatedRecord.TotalEpisodes = len(result.Detail.Episodes)
			updatedRecord.Title = result.Detail.Title
			updatedRecord.Cover = result.Detail.Poster
			updatedRecord.Year = result.Detail.Year

			if err := database.UpdatePlayRecord(&updatedRecord); err != nil {
				log.Printf("更新播放记录失败 [%s]: %v", key, err)
			} else {
				log.Printf("播放记录更新成功 [%s]", key)
				updatedCount++
			}
		}
	}

	log.Printf("播放记录更新完成，共更新 %d 条记录", updatedCount)
}

// updateFavorites 更新收藏夹记录
func updateFavorites(favorites map[string]models.Favorite, detailResults map[string]VideoDetailResult) {
	log.Println("开始更新收藏夹记录...")
	updatedCount := 0

	for key, favorite := range favorites {
		result, exists := detailResults[key]
		if !exists {
			log.Printf("未找到收藏夹记录 [%s] 的详情结果", key)
			continue
		}

		if result.Error != nil {
			log.Printf("获取视频详情失败 [%s]: %v", key, result.Error)
			continue
		}

		if result.Detail == nil {
			log.Printf("视频详情为空 [%s]", key)
			continue
		}

		// 检查 total_episodes 是否不同
		if len(result.Detail.Episodes) != favorite.TotalEpisodes {
			log.Printf("发现集数变化 [%s]: %d -> %d", key, favorite.TotalEpisodes, len(result.Detail.Episodes))

			// 更新收藏夹记录（只更新 total_episodes、year、cover 和 title）
			updatedFavorite := favorite
			updatedFavorite.TotalEpisodes = len(result.Detail.Episodes)
			updatedFavorite.Title = result.Detail.Title
			updatedFavorite.Cover = result.Detail.Poster
			updatedFavorite.Year = result.Detail.Year

			if err := database.UpdateFavorite(&updatedFavorite); err != nil {
				log.Printf("更新收藏夹记录失败 [%s]: %v", key, err)
			} else {
				log.Printf("收藏夹记录更新成功 [%s]", key)
				updatedCount++
			}
		}
	}

	log.Printf("收藏夹记录更新完成，共更新 %d 条记录", updatedCount)
}

// GetName 获取任务名称
func (h *HourlyTask) GetName() string {
	return h.name
}
