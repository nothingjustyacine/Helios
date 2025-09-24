package lib

import (
	"regexp"
	"strings"
)

// cleanHtmlTags 清理 HTML 标签
func CleanHtmlTags(html string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	return strings.TrimSpace(re.ReplaceAllString(html, ""))
}

// extractYear 从年份字符串中提取四位数字年份
func ExtractYear(yearStr string) string {
	if yearStr == "" {
		return "unknown"
	}

	re := regexp.MustCompile(`\d{4}`)
	if matches := re.FindString(yearStr); matches != "" {
		return matches
	}

	return "unknown"
}

// cleanTitle 清理标题，去除多余空格
func CleanTitle(title string) string {
	title = strings.TrimSpace(title)
	re := regexp.MustCompile(`\s+`)
	return re.ReplaceAllString(title, " ")
}

// extractEpisodes 从播放URL中提取集数信息
func ExtractEpisodes(vodPlayURL string) ([]string, []string) {
	if vodPlayURL == "" {
		return []string{}, []string{}
	}

	var episodes []string
	var titles []string

	// 先用 $$$ 分割
	vodPlayURLArray := strings.Split(vodPlayURL, "$$$")

	// 分集之间#分割，标题和播放链接 $ 分割
	for _, url := range vodPlayURLArray {
		var matchEpisodes []string
		var matchTitles []string
		titleURLArray := strings.Split(url, "#")

		for _, titleURL := range titleURLArray {
			episodeTitleURL := strings.Split(titleURL, "$")
			if len(episodeTitleURL) == 2 && strings.HasSuffix(episodeTitleURL[1], ".m3u8") {
				matchTitles = append(matchTitles, episodeTitleURL[0])
				matchEpisodes = append(matchEpisodes, episodeTitleURL[1])
			}
		}

		if len(matchEpisodes) > len(episodes) {
			episodes = matchEpisodes
			titles = matchTitles
		}
	}

	return episodes, titles
}
