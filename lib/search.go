package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"helios/config"
	"helios/models"
)

// ApiSearchItem 表示从 API 返回的搜索项结构
type ApiSearchItem struct {
	VodID       int     `json:"vod_id"`
	VodName     string  `json:"vod_name"`
	VodPic      string  `json:"vod_pic"`
	VodRemarks  *string `json:"vod_remarks,omitempty"`
	VodPlayURL  *string `json:"vod_play_url,omitempty"`
	VodClass    *string `json:"vod_class,omitempty"`
	VodYear     *string `json:"vod_year,omitempty"`
	VodContent  *string `json:"vod_content,omitempty"`
	VodDoubanID *int    `json:"vod_douban_id,omitempty"`
	TypeName    *string `json:"type_name,omitempty"`
}

type ApiSearchItem2 struct {
	VodID       string  `json:"vod_id"`
	VodName     string  `json:"vod_name"`
	VodPic      string  `json:"vod_pic"`
	VodRemarks  *string `json:"vod_remarks,omitempty"`
	VodPlayURL  *string `json:"vod_play_url,omitempty"`
	VodClass    *string `json:"vod_class,omitempty"`
	VodYear     *string `json:"vod_year,omitempty"`
	VodContent  *string `json:"vod_content,omitempty"`
	VodDoubanID *int    `json:"vod_douban_id,omitempty"`
	TypeName    *string `json:"type_name,omitempty"`
}

// models.SearchResult 搜索结果数据结构

// CachedSearchPage 缓存的搜索页面数据
type CachedSearchPage struct {
	Status    string                `json:"status"` // "ok", "forbidden", "timeout"
	Data      []models.SearchResult `json:"data"`
	PageCount *int                  `json:"page_count,omitempty"`
	Timestamp time.Time             `json:"timestamp"`
	ExpiresAt time.Time             `json:"expires_at"` // 过期时间
}

// 全局变量
var (
	// API 搜索配置
	searchPath     = "?ac=videolist&wd="
	searchPagePath = "?ac=videolist&wd={query}&pg={page}"
	searchHeaders  = map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
		"Accept":     "application/json",
	}

	// API 详情配置
	detailPath    = "?ac=videolist&ids="
	detailHeaders = map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
		"Accept":     "application/json",
	}

	// M3U8 链接匹配模式
	m3u8Pattern = regexp.MustCompile(`\$?(https?://[^"'\s]+?\.m3u8)`)
	ffzyPattern = regexp.MustCompile(`\$(https?://[^"'\s]+?/\d{8}/\d+_[a-f0-9]+/index\.m3u8)`)
)

// searchWithCache 通用的带缓存搜索函数
func SearchWithCache(apiSite models.APISite, query string, page int, url string, timeoutMs int) ([]models.SearchResult, *int, error) {
	// 先查缓存
	cached := GetCachedSearchPage(apiSite.Key, query, page)
	if cached != nil {
		if cached.Status == "ok" {
			return cached.Data, cached.PageCount, nil
		} else {
			return []models.SearchResult{}, nil, nil
		}
	}

	// 缓存未命中，发起网络请求
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return []models.SearchResult{}, nil, err
	}

	// 设置请求头
	for key, value := range searchHeaders {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		// 检查是否是超时错误
		if ctx.Err() == context.DeadlineExceeded {
			SetCachedSearchPage(apiSite.Key, query, page, "timeout", []models.SearchResult{}, nil)
		}
		fmt.Println("搜索请求失败", err)
		return []models.SearchResult{}, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 {
		SetCachedSearchPage(apiSite.Key, query, page, "forbidden", []models.SearchResult{}, nil)
		// fmt.Println("搜索请求失败", resp.StatusCode)
		return []models.SearchResult{}, nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		// fmt.Println("搜索请求失败", resp.StatusCode)
		return []models.SearchResult{}, nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		// fmt.Println("搜索请求失败", err)
		return []models.SearchResult{}, nil, err
	}

	// 尝试第一种数据结构 (ApiSearchItem with int VodID)
	var data struct {
		List      []ApiSearchItem `json:"list"`
		PageCount int             `json:"pagecount"`
	}

	var data2 struct {
		List      []ApiSearchItem2 `json:"list"`
		PageCount int              `json:"pagecount"`
	}

	var items []ApiSearchItem
	if err := json.Unmarshal(body, &data); err != nil {
		// 如果第一种结构解析失败，尝试第二种结构 (ApiSearchItem2 with string VodID)
		if err2 := json.Unmarshal(body, &data2); err2 != nil {
			// fmt.Println("搜索请求失败，两种数据结构都解析失败", err, err2)
			fmt.Println(string(body))
			return []models.SearchResult{}, nil, err
		}
		// 将 ApiSearchItem2 转换为 ApiSearchItem
		items = make([]ApiSearchItem, len(data2.List))
		for i, item2 := range data2.List {
			// 将 string 类型的 VodID 转换为 int
			vodID, err := strconv.Atoi(item2.VodID)
			if err != nil {
				// 如果转换失败，跳过这个项目或使用默认值
				// fmt.Printf("警告: 无法转换 VodID %s 为整数，跳过该项目\n", item2.VodID)
				continue
			}
			items[i] = ApiSearchItem{
				VodID:       vodID,
				VodName:     item2.VodName,
				VodPic:      item2.VodPic,
				VodRemarks:  item2.VodRemarks,
				VodPlayURL:  item2.VodPlayURL,
				VodClass:    item2.VodClass,
				VodYear:     item2.VodYear,
				VodContent:  item2.VodContent,
				VodDoubanID: item2.VodDoubanID,
				TypeName:    item2.TypeName,
			}
		}
	} else {
		items = data.List
	}

	if len(items) == 0 {
		// 空结果不做负缓存要求
		// fmt.Println("搜索请求失败", "空结果")
		return []models.SearchResult{}, nil, nil
	}

	// 统一处理结果数据
	var allResults []models.SearchResult
	for _, item := range items {
		var episodes []string
		var titles []string

		// 使用工具函数从 vod_play_url 提取 m3u8 链接
		if item.VodPlayURL != nil && *item.VodPlayURL != "" {
			episodes, titles = ExtractEpisodes(*item.VodPlayURL)
		}

		// 提取年份
		year := "unknown"
		if item.VodYear != nil && *item.VodYear != "" {
			year = ExtractYear(*item.VodYear)
		}

		// 清理描述
		var desc *string
		if item.VodContent != nil && *item.VodContent != "" {
			cleaned := CleanHtmlTags(*item.VodContent)
			desc = &cleaned
		}

		// 清理标题
		title := CleanTitle(item.VodName)

		result := models.SearchResult{
			ID:             strconv.Itoa(item.VodID),
			Title:          title,
			Poster:         item.VodPic,
			Episodes:       episodes,
			EpisodesTitles: titles,
			Source:         apiSite.Key,
			SourceName:     apiSite.Name,
			Class:          item.VodClass,
			Year:           year,
			Desc:           desc,
			TypeName:       item.TypeName,
			DoubanID:       item.VodDoubanID,
		}

		allResults = append(allResults, result)
	}

	// 过滤掉集数为 0 的结果
	var results []models.SearchResult
	for _, result := range allResults {
		if len(result.Episodes) > 0 {
			results = append(results, result)
		}
	}

	var pageCount *int
	if page == 1 && data.PageCount > 0 {
		pageCount = &data.PageCount
	}

	// 写入缓存（成功）
	SetCachedSearchPage(apiSite.Key, query, page, "ok", results, pageCount)
	return results, pageCount, nil
}

// SearchFromAPI 从 API 搜索
func SearchFromAPI(apiSite models.APISite, query string, maxPages int) ([]models.SearchResult, error) {
	apiBaseURL := apiSite.API
	apiURL := apiBaseURL + searchPath + query

	// 使用新的缓存搜索函数处理第一页
	firstPageResults, pageCountFromFirst, err := SearchWithCache(apiSite, query, 1, apiURL, 8000)
	if err != nil {
		// fmt.Println("搜索请求失败", err)
		return []models.SearchResult{}, err
	}

	results := firstPageResults
	pageCount := 1
	if pageCountFromFirst != nil {
		pageCount = *pageCountFromFirst
	}

	// 确定需要获取的额外页数
	pagesToFetch := pageCount - 1
	if pagesToFetch > maxPages-1 {
		pagesToFetch = maxPages - 1
	}

	// 如果有额外页数，获取更多页的结果
	if pagesToFetch > 0 {
		var wg sync.WaitGroup
		var mutex sync.Mutex

		for page := 2; page <= pagesToFetch+1; page++ {
			wg.Add(1)
			go func(pageNum int) {
				defer wg.Done()

				pageURL := apiBaseURL + strings.ReplaceAll(
					strings.ReplaceAll(searchPagePath, "{query}", query),
					"{page}", fmt.Sprintf("%d", pageNum),
				)

				pageResults, _, err := SearchWithCache(apiSite, query, pageNum, pageURL, 8000)
				if err == nil && len(pageResults) > 0 {
					mutex.Lock()
					results = append(results, pageResults...)
					mutex.Unlock()
				}
			}(page)
		}

		wg.Wait()
	}

	return results, nil
}

// yellowWords 黄色过滤关键词列表
var yellowWords = []string{
	"伦理片",
	"福利",
	"里番动漫",
	"门事件",
	"萝莉少女",
	"制服诱惑",
	"国产传媒",
	"cosplay",
	"黑丝诱惑",
	"无码",
	"日本无码",
	"有码",
	"日本有码",
	"SWAG",
	"网红主播",
	"色情片",
	"同性片",
	"福利视频",
	"福利片",
	"写真热舞",
	"倫理片",
	"理论片",
	"韩国伦理",
	"港台三级",
	"电影解说",
	"伦理",
	"日本伦理",
}

// FilterYellowContent 过滤黄色内容
func FilterYellowContent(results []models.SearchResult) []models.SearchResult {
	var filteredResults []models.SearchResult

	for _, result := range results {
		if result.TypeName != nil {
			typeName := *result.TypeName
			shouldFilter := false

			for _, word := range yellowWords {
				if strings.Contains(typeName, word) {
					shouldFilter = true
					break
				}
			}

			if !shouldFilter {
				filteredResults = append(filteredResults, result)
			}
		} else {
			// 如果没有 type_name，不过滤
			filteredResults = append(filteredResults, result)
		}
	}

	return filteredResults
}

// ApiDetailItem 表示从 API 返回的详情项结构
type ApiDetailItem struct {
	VodID       int     `json:"vod_id"`
	VodName     string  `json:"vod_name"`
	VodPic      string  `json:"vod_pic"`
	VodPlayURL  *string `json:"vod_play_url,omitempty"`
	VodClass    *string `json:"vod_class,omitempty"`
	VodYear     *string `json:"vod_year,omitempty"`
	VodContent  *string `json:"vod_content,omitempty"`
	VodDoubanID *int    `json:"vod_douban_id,omitempty"`
	TypeName    *string `json:"type_name,omitempty"`
}

// ApiDetailResponse API 详情响应结构
type ApiDetailResponse struct {
	List []ApiDetailItem `json:"list"`
}

// GetDetailFromAPI 从 API 获取详情
func GetDetailFromAPI(apiSite models.APISite, id string) (*models.SearchResult, error) {
	// 检查是否有特殊详情处理
	if apiSite.Detail != "" {
		return handleSpecialSourceDetail(id, apiSite)
	}

	// 构建详情 URL
	detailURL := apiSite.API + detailPath + id

	// 创建带超时的请求
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", detailURL, nil)
	if err != nil {
		// fmt.Println("详情请求失败", err)
		return nil, err
	}

	// 设置请求头
	for key, value := range detailHeaders {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("详情请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("详情请求失败: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data ApiDetailResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	if len(data.List) == 0 {
		return nil, fmt.Errorf("获取到的详情内容无效")
	}

	videoDetail := data.List[0]
	var episodes []string
	var titles []string

	// 处理播放源拆分
	if videoDetail.VodPlayURL != nil && *videoDetail.VodPlayURL != "" {
		episodes, titles = ExtractEpisodes(*videoDetail.VodPlayURL)
	}

	// 如果播放源为空，则尝试从内容中解析 m3u8
	if len(episodes) == 0 && videoDetail.VodContent != nil && *videoDetail.VodContent != "" {
		matches := m3u8Pattern.FindAllString(*videoDetail.VodContent, -1)
		for _, match := range matches {
			// 去掉开头的 $ 符号
			cleanMatch := strings.TrimPrefix(match, "$")
			episodes = append(episodes, cleanMatch)
		}
	}

	// 提取年份
	year := "unknown"
	if videoDetail.VodYear != nil && *videoDetail.VodYear != "" {
		year = ExtractYear(*videoDetail.VodYear)
	}

	// 清理描述
	var desc *string
	if videoDetail.VodContent != nil && *videoDetail.VodContent != "" {
		cleaned := CleanHtmlTags(*videoDetail.VodContent)
		desc = &cleaned
	}

	// 清理标题
	title := CleanTitle(videoDetail.VodName)

	result := &models.SearchResult{
		ID:             id,
		Title:          title,
		Poster:         videoDetail.VodPic,
		Episodes:       episodes,
		EpisodesTitles: titles,
		Source:         apiSite.Key,
		SourceName:     apiSite.Name,
		Class:          videoDetail.VodClass,
		Year:           year,
		Desc:           desc,
		TypeName:       videoDetail.TypeName,
		DoubanID:       videoDetail.VodDoubanID,
	}

	return result, nil
}

// handleSpecialSourceDetail 处理特殊源的详情
func handleSpecialSourceDetail(id string, apiSite models.APISite) (*models.SearchResult, error) {
	detailURL := apiSite.Detail + "/index.php/vod/detail/id/" + id + ".html"

	// 创建带超时的请求
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", detailURL, nil)
	if err != nil {
		return nil, err
	}

	// 设置请求头
	for key, value := range detailHeaders {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("详情页请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("详情页请求失败: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	html := string(body)
	var matches []string

	// 根据不同的源使用不同的匹配模式
	if apiSite.Key == "ffzy" {
		matches = ffzyPattern.FindAllString(html, -1)
	}

	// 如果没有匹配到，使用通用模式
	if len(matches) == 0 {
		matches = m3u8Pattern.FindAllString(html, -1)
	}

	// 去重并清理链接前缀
	var cleanMatches []string
	seen := make(map[string]bool)
	for _, match := range matches {
		// 去掉开头的 $
		cleanMatch := strings.TrimPrefix(match, "$")
		// 去掉括号后的内容
		if parenIndex := strings.Index(cleanMatch, "("); parenIndex > 0 {
			cleanMatch = cleanMatch[:parenIndex]
		}
		if !seen[cleanMatch] {
			seen[cleanMatch] = true
			cleanMatches = append(cleanMatches, cleanMatch)
		}
	}

	// 根据 matches 数量生成剧集标题
	var episodesTitles []string
	for i := 1; i <= len(cleanMatches); i++ {
		episodesTitles = append(episodesTitles, fmt.Sprintf("%d", i))
	}

	// 提取标题
	titlePattern := regexp.MustCompile(`<h1[^>]*>([^<]+)</h1>`)
	titleMatch := titlePattern.FindStringSubmatch(html)
	titleText := ""
	if len(titleMatch) > 1 {
		titleText = strings.TrimSpace(titleMatch[1])
	}

	// 提取描述
	descPattern := regexp.MustCompile(`<div[^>]*class=["']sketch["'][^>]*>([\s\S]*?)</div>`)
	descMatch := descPattern.FindStringSubmatch(html)
	descText := ""
	if len(descMatch) > 1 {
		descText = CleanHtmlTags(descMatch[1])
	}

	// 提取封面
	coverPattern := regexp.MustCompile(`(https?://[^"'\s]+?\.jpg)`)
	coverMatch := coverPattern.FindString(html)
	coverURL := ""
	if coverMatch != "" {
		coverURL = strings.TrimSpace(coverMatch)
	}

	// 提取年份
	yearPattern := regexp.MustCompile(`>(\d{4})<`)
	yearMatch := yearPattern.FindStringSubmatch(html)
	yearText := "unknown"
	if len(yearMatch) > 1 {
		yearText = yearMatch[1]
	}

	result := &models.SearchResult{
		ID:             id,
		Title:          titleText,
		Poster:         coverURL,
		Episodes:       cleanMatches,
		EpisodesTitles: episodesTitles,
		Source:         apiSite.Key,
		SourceName:     apiSite.Name,
		Class:          nil,
		Year:           yearText,
		Desc:           &descText,
		TypeName:       nil,
		DoubanID:       nil,
	}

	return result, nil
}

// FetchVideoDetailOptions 获取视频详情的选项
type FetchVideoDetailOptions struct {
	Source        string
	ID            string
	FallbackTitle string
}

// FetchVideoDetail 获取视频详情
// 优先通过搜索接口查找精确匹配，如果找不到则调用详情接口
func FetchVideoDetail(options FetchVideoDetailOptions) (*models.SearchResult, error) {
	// 获取 API 站点配置
	apiSites := config.GlobalConfig.APISites
	apiSite, exists := apiSites[options.Source]
	if !exists {
		// fmt.Println("无效的API来源", options.Source)
		return nil, fmt.Errorf("无效的API来源")
	}

	// 如果有 fallbackTitle，先尝试通过搜索接口查找精确匹配
	if options.FallbackTitle != "" {
		searchResults, err := SearchFromAPI(apiSite, options.FallbackTitle, 5)
		if err == nil {
			// 查找精确匹配
			for _, item := range searchResults {
				if item.Source == options.Source && item.ID == options.ID {
					return &item, nil
				}
			}
		}
		// 如果搜索失败或没找到匹配项，继续执行详情接口调用
	}

	// 调用详情接口
	detail, err := GetDetailFromAPI(apiSite, options.ID)
	if err != nil {
		return nil, fmt.Errorf("获取视频详情失败: %v", err)
	}

	return detail, nil
}
