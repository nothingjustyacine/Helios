package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"helios/models"

	"github.com/bytedance/sonic"
	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func InitializeDatabase(dbPath string) error {
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %v", err)
	}

	_, err := os.Stat(dbPath)
	dbExists := err == nil

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}

	if !dbExists {
		log.Println("Database does not exist, creating new database...")
		if err := initializeDatabaseSchema(db); err != nil {
			db.Close()
			return fmt.Errorf("failed to initialize database schema: %v", err)
		}
		log.Println("Database schema initialized successfully")
	} else {
		log.Println("Existing database found, skipping initialization")
	}

	DB = db
	log.Printf("Database connection established: %s", dbPath)
	return nil
}

func initializeDatabaseSchema(db *sql.DB) error {
	log.Println("Executing database initialization...")

	err := initializeDatabaseTables(db)
	if err != nil {
		return fmt.Errorf("failed to create tables: %v", err)
	}

	return nil
}

func initializeDatabaseTables(db *sql.DB) error {
	createFavoritesTable := `
	CREATE TABLE IF NOT EXISTS favorites (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		source TEXT NOT NULL,
		source_id TEXT NOT NULL,
		source_name TEXT NOT NULL,
		total_episodes INTEGER NOT NULL,
		title TEXT NOT NULL,
		year TEXT NOT NULL,
		cover TEXT NOT NULL,
		save_time INTEGER NOT NULL,
		search_title TEXT NOT NULL
	);`

	createSearchHistoryTable := `
	CREATE TABLE IF NOT EXISTS search_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		record TEXT NOT NULL
	);`

	createPlayRecordsTable := `
	CREATE TABLE IF NOT EXISTS play_records (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		source TEXT NOT NULL,
		source_id TEXT NOT NULL,
		source_name TEXT NOT NULL,
		cover TEXT NOT NULL,
		year TEXT NOT NULL,
		index_number INTEGER NOT NULL,
		total_episodes INTEGER NOT NULL,
		play_time INTEGER NOT NULL,
		total_time INTEGER NOT NULL,
		save_time INTEGER NOT NULL,
		search_title TEXT NOT NULL
	);`

	tables := []string{
		createFavoritesTable,
		createSearchHistoryTable,
		createPlayRecordsTable,
	}

	for _, table := range tables {
		if _, err := db.Exec(table); err != nil {
			return fmt.Errorf("failed to create table: %v", err)
		}
	}

	// 创建索引
	if err := createIndexes(db); err != nil {
		return fmt.Errorf("failed to create indexes: %v", err)
	}

	return nil
}

func createIndexes(db *sql.DB) error {
	// 为 favorites 表创建 source + source_id 唯一索引
	createFavoritesUniqueIndex := `
	CREATE UNIQUE INDEX IF NOT EXISTS idx_favorites_source_source_id_unique 
	ON favorites (source, source_id);`

	// 为 play_records 表创建 source + source_id 唯一索引
	createPlayRecordsUniqueIndex := `
	CREATE UNIQUE INDEX IF NOT EXISTS idx_play_records_source_source_id_unique 
	ON play_records (source, source_id);`

	indexes := []string{
		createFavoritesUniqueIndex,
		createPlayRecordsUniqueIndex,
	}

	for _, index := range indexes {
		if _, err := db.Exec(index); err != nil {
			return fmt.Errorf("failed to create index: %v", err)
		}
	}

	return nil
}

func CloseDatabase() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

// PlayRecordExists 检查播放记录是否存在
func PlayRecordExists(source, sourceID string) (bool, error) {
	var count int
	query := "SELECT COUNT(*) FROM play_records WHERE source = ? AND source_id = ?"
	err := DB.QueryRow(query, source, sourceID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// InsertPlayRecord 插入新的播放记录
func InsertPlayRecord(record *models.PlayRecord) error {
	query := `
		INSERT INTO play_records (title, source, source_id, source_name, cover, year, 
			index_number, total_episodes, play_time, total_time, save_time, search_title)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := DB.Exec(query,
		record.Title, record.Source, record.SourceID, record.SourceName,
		record.Cover, record.Year, record.IndexNumber, record.TotalEpisodes,
		record.PlayTime, record.TotalTime, record.SaveTime, record.SearchTitle)
	return err
}

// UpdatePlayRecord 更新现有的播放记录
func UpdatePlayRecord(record *models.PlayRecord) error {
	query := `
		UPDATE play_records SET 
			title = ?, cover = ?, year = ?, total_episodes = ?
		WHERE source = ? AND source_id = ?`

	_, err := DB.Exec(query,
		record.Title, record.Cover, record.Year, record.TotalEpisodes,
		record.Source, record.SourceID)
	return err
}

// GetAllPlayRecords 获取所有播放记录
func GetAllPlayRecords() (map[string]models.PlayRecord, error) {
	query := `
		SELECT title, source, source_id, source_name, cover, year, 
			index_number, total_episodes, play_time, total_time, save_time, search_title
		FROM play_records`

	rows, err := DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make(map[string]models.PlayRecord)

	for rows.Next() {
		var record models.PlayRecord
		err := rows.Scan(
			&record.Title, &record.Source, &record.SourceID, &record.SourceName,
			&record.Cover, &record.Year, &record.IndexNumber, &record.TotalEpisodes,
			&record.PlayTime, &record.TotalTime, &record.SaveTime, &record.SearchTitle,
		)
		if err != nil {
			return nil, err
		}

		// 使用 source+source_id 作为 key
		key := record.Source + "+" + record.SourceID
		records[key] = record
	}

	return records, nil
}

// UpsertPlayRecord 使用 UPSERT 操作插入或更新播放记录
func UpsertPlayRecord(record *models.PlayRecord) error {
	query := `
		INSERT INTO play_records (title, source, source_id, source_name, cover, year, 
			index_number, total_episodes, play_time, total_time, save_time, search_title)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(source, source_id) DO UPDATE SET
			title = excluded.title,
			source_name = excluded.source_name,
			cover = excluded.cover,
			year = excluded.year,
			index_number = excluded.index_number,
			total_episodes = excluded.total_episodes,
			play_time = excluded.play_time,
			total_time = excluded.total_time,
			save_time = excluded.save_time,
			search_title = excluded.search_title`

	_, err := DB.Exec(query,
		record.Title, record.Source, record.SourceID, record.SourceName,
		record.Cover, record.Year, record.IndexNumber, record.TotalEpisodes,
		record.PlayTime, record.TotalTime, record.SaveTime, record.SearchTitle)
	return err
}

// DeletePlayRecord 根据 source 和 source_id 删除播放记录
func DeletePlayRecord(source, sourceID string) error {
	query := "DELETE FROM play_records WHERE source = ? AND source_id = ?"
	_, err := DB.Exec(query, source, sourceID)
	return err
}

// GetAllFavorites 获取所有收藏夹记录
func GetAllFavorites() (map[string]models.Favorite, error) {
	query := `
		SELECT source, source_id, source_name, total_episodes, title, year, 
			cover, save_time, search_title
		FROM favorites`

	rows, err := DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	favorites := make(map[string]models.Favorite)

	for rows.Next() {
		var favorite models.Favorite
		err := rows.Scan(
			&favorite.Source, &favorite.SourceID, &favorite.SourceName,
			&favorite.TotalEpisodes, &favorite.Title, &favorite.Year,
			&favorite.Cover, &favorite.SaveTime, &favorite.SearchTitle,
		)
		if err != nil {
			return nil, err
		}

		// 使用 source+source_id 作为 key
		key := favorite.Source + "+" + favorite.SourceID
		favorites[key] = favorite
	}

	return favorites, nil
}

// InsertFavorite 插入新的收藏夹记录
func InsertFavorite(favorite *models.Favorite) error {
	query := `
		INSERT INTO favorites (source, source_id, source_name, total_episodes, 
			title, year, cover, save_time, search_title)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := DB.Exec(query,
		favorite.Source, favorite.SourceID, favorite.SourceName,
		favorite.TotalEpisodes, favorite.Title, favorite.Year,
		favorite.Cover, favorite.SaveTime, favorite.SearchTitle)
	return err
}

// UpdateFavorite 更新现有的收藏夹记录
func UpdateFavorite(favorite *models.Favorite) error {
	query := `
		UPDATE favorites SET 
			total_episodes = ?, title = ?, year = ?, cover = ?
		WHERE source = ? AND source_id = ?`

	_, err := DB.Exec(query,
		favorite.TotalEpisodes, favorite.Title, favorite.Year, favorite.Cover,
		favorite.Source, favorite.SourceID)
	return err
}

// DeleteFavorite 根据 source 和 source_id 删除收藏夹记录
func DeleteFavorite(source, sourceID string) error {
	query := "DELETE FROM favorites WHERE source = ? AND source_id = ?"
	_, err := DB.Exec(query, source, sourceID)
	return err
}

// UpsertFavorite 使用 UPSERT 操作插入或更新收藏夹记录
func UpsertFavorite(favorite *models.Favorite) error {
	query := `
		INSERT INTO favorites (source, source_id, source_name, total_episodes, 
			title, year, cover, save_time, search_title)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(source, source_id) DO UPDATE SET
			source_name = excluded.source_name,
			total_episodes = excluded.total_episodes,
			title = excluded.title,
			year = excluded.year,
			cover = excluded.cover,
			save_time = excluded.save_time,
			search_title = excluded.search_title`

	_, err := DB.Exec(query,
		favorite.Source, favorite.SourceID, favorite.SourceName,
		favorite.TotalEpisodes, favorite.Title, favorite.Year,
		favorite.Cover, favorite.SaveTime, favorite.SearchTitle)
	return err
}

// GetSearchHistory 获取搜索历史记录
func GetSearchHistory() (string, error) {
	var record string
	query := "SELECT record FROM search_history WHERE id = 1"
	err := DB.QueryRow(query).Scan(&record)
	if err != nil {
		if err == sql.ErrNoRows {
			return "[]", nil // 没有记录时返回空数组JSON
		}
		return "", err
	}
	return record, nil
}

// UpsertSearchHistory 更新搜索历史记录
func UpsertSearchHistory(keywords []string) error {
	// 开始事务
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 将关键词数组转换为JSON字符串
	jsonData, err := sonic.MarshalString(keywords)
	if err != nil {
		return err
	}

	// 检查是否存在 id=1 的记录
	var count int
	err = tx.QueryRow("SELECT COUNT(*) FROM search_history WHERE id = 1").Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		// 如果不存在记录，插入新记录，强制 id=1
		_, err = tx.Exec("INSERT INTO search_history (id, record) VALUES (1, ?)", jsonData)
		if err != nil {
			return err
		}
	} else {
		// 如果存在记录，更新 id=1 的记录
		_, err = tx.Exec("UPDATE search_history SET record = ? WHERE id = 1", jsonData)
		if err != nil {
			return err
		}
	}

	// 提交事务
	return tx.Commit()
}

// UpdateSearchHistoryInTransaction 在事务中更新搜索历史并返回更新后的关键词数组
func UpdateSearchHistoryInTransaction(keyword string) ([]string, error) {
	// 开始事务
	tx, err := DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 获取现有的搜索历史
	var record string
	query := "SELECT record FROM search_history WHERE id = 1"
	err = tx.QueryRow(query).Scan(&record)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	var keywords []string
	if record != "" {
		// 解析现有的JSON数组
		if err := sonic.UnmarshalString(record, &keywords); err != nil {
			return nil, err
		}
	}

	// 检查关键词是否已存在
	keywordIndex := -1
	for i, k := range keywords {
		if k == keyword {
			keywordIndex = i
			break
		}
	}

	if keywordIndex == -1 {
		// 关键词不存在，添加到开头
		keywords = append([]string{keyword}, keywords...)
	} else {
		// 关键词存在，移动到开头
		// 先移除现有位置的关键词
		keywords = append(keywords[:keywordIndex], keywords[keywordIndex+1:]...)
		// 添加到开头
		keywords = append([]string{keyword}, keywords...)
	}

	// 将关键词数组转换为JSON字符串
	jsonData, err := sonic.MarshalString(keywords)
	if err != nil {
		return nil, err
	}

	// 检查是否存在 id=1 的记录
	var count int
	err = tx.QueryRow("SELECT COUNT(*) FROM search_history WHERE id = 1").Scan(&count)
	if err != nil {
		return nil, err
	}

	if count == 0 {
		// 如果不存在记录，插入新记录，强制 id=1
		_, err = tx.Exec("INSERT INTO search_history (id, record) VALUES (1, ?)", jsonData)
		if err != nil {
			return nil, err
		}
	} else {
		// 如果存在记录，更新 id=1 的记录
		_, err = tx.Exec("UPDATE search_history SET record = ? WHERE id = 1", jsonData)
		if err != nil {
			return nil, err
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return keywords, nil
}

// DeleteSearchHistoryKeyword 从搜索历史中删除指定关键词
func DeleteSearchHistoryKeyword(keyword string) error {
	// 开始事务
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 获取现有的搜索历史
	var record string
	query := "SELECT record FROM search_history WHERE id = 1"
	err = tx.QueryRow(query).Scan(&record)
	if err != nil {
		if err == sql.ErrNoRows {
			// 没有记录，返回空数组
			return nil
		}
		return err
	}

	var keywords []string
	if record != "" {
		// 解析现有的JSON数组
		if err := sonic.UnmarshalString(record, &keywords); err != nil {
			return err
		}
	}

	// 查找并删除指定关键词
	newKeywords := make([]string, 0, len(keywords))
	for _, k := range keywords {
		if k != keyword {
			newKeywords = append(newKeywords, k)
		}
	}

	// 将更新后的关键词数组转换为JSON字符串
	jsonData, err := sonic.MarshalString(newKeywords)
	if err != nil {
		return err
	}

	// 更新数据库记录
	_, err = tx.Exec("UPDATE search_history SET record = ? WHERE id = 1", jsonData)
	if err != nil {
		return err
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}
