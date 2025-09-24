package models

import "time"

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthInfo struct {
	Role     string `json:"role"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type SearchRequest struct {
	Query string `json:"query"`
	Type  string `json:"type"`
}

type SearchResult struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Poster         string   `json:"poster"`
	Episodes       []string `json:"episodes"`
	EpisodesTitles []string `json:"episodes_titles"`
	Source         string   `json:"source"`
	SourceName     string   `json:"source_name"`
	Class          *string  `json:"class,omitempty"`
	Year           string   `json:"year"`
	Desc           *string  `json:"desc,omitempty"`
	TypeName       *string  `json:"type_name,omitempty"`
	DoubanID       *int     `json:"douban_id,omitempty"`
}

type DetailRequest struct {
	ID string `json:"id"`
}

type DetailResponse struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Type        string    `json:"type"`
	CreatedAt   time.Time `json:"created_at"`
}

type FavoritesRequest struct {
	UserID string `json:"user_id"`
}

type FavoritePostRequest struct {
	Key      string   `json:"key"`
	Favorite Favorite `json:"favorite"`
}

type FavoritesResponse struct {
	Favorites []Favorite `json:"favorites"`
	Total     int        `json:"total"`
}

type Favorite struct {
	Source        string `json:"source" db:"source"`
	SourceID      string `json:"source_id" db:"source_id"`
	SourceName    string `json:"source_name" db:"source_name"`
	TotalEpisodes int    `json:"total_episodes" db:"total_episodes"`
	Title         string `json:"title" db:"title"`
	Year          string `json:"year" db:"year"`
	Cover         string `json:"cover" db:"cover"`
	SaveTime      int64  `json:"save_time" db:"save_time"`
	SearchTitle   string `json:"search_title" db:"search_title"`
	Origin        string `json:"origin,omitempty"`
}

type SearchHistoryRequest struct {
	Keyword string `json:"keyword"`
}

type SearchHistoryResponse struct {
	History []SearchRecord `json:"history"`
	Total   int            `json:"total"`
}

type SearchRecord struct {
	Record string `json:"record" db:"record"`
}

type PlayRecordsRequest struct {
	UserID string `json:"user_id"`
}

type PlayRecordPostRequest struct {
	Key    string     `json:"key"`
	Record PlayRecord `json:"record"`
}

type PlayRecordsResponse struct {
	Records []PlayRecord `json:"records"`
	Total   int          `json:"total"`
}

type PlayRecord struct {
	Title         string `json:"title" db:"title"`
	Source        string `json:"source" db:"source"`
	SourceID      string `json:"source_id" db:"source_id"`
	SourceName    string `json:"source_name" db:"source_name"`
	Cover         string `json:"cover" db:"cover"`
	Year          string `json:"year" db:"year"`
	IndexNumber   int    `json:"index" db:"index_number"`
	TotalEpisodes int    `json:"total_episodes" db:"total_episodes"`
	PlayTime      int    `json:"play_time" db:"play_time"`
	TotalTime     int    `json:"total_time" db:"total_time"`
	SaveTime      int64  `json:"save_time" db:"save_time"`
	SearchTitle   string `json:"search_title" db:"search_title"`
}

// SSE 事件相关结构体
type SSEStartEvent struct {
	Type         string `json:"type"`
	Query        string `json:"query"`
	TotalSources int    `json:"totalSources"`
	Timestamp    int64  `json:"timestamp"`
}

type SSESourceResultEvent struct {
	Type       string         `json:"type"`
	Source     string         `json:"source"`
	SourceName string         `json:"sourceName"`
	Results    []SearchResult `json:"results"`
	Timestamp  int64          `json:"timestamp"`
}

type SSESourceErrorEvent struct {
	Type       string `json:"type"`
	Source     string `json:"source"`
	SourceName string `json:"sourceName"`
	Error      string `json:"error"`
	Timestamp  int64  `json:"timestamp"`
}

type SSECompleteEvent struct {
	Type             string `json:"type"`
	TotalResults     int    `json:"totalResults"`
	CompletedSources int    `json:"completedSources"`
	Timestamp        int64  `json:"timestamp"`
}
