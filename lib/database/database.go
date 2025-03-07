package database

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/robfig/cron/v3"
	"github.com/tpc3/Bocchi-Re/lib/config"
)

// UsageRecord represents a row in the usage_records table
type UsageRecord struct {
	ID         int
	ServerID   string
	Model      string
	UsageType  string
	UsageCount int
	Month      int
	Year       int
	CreatedAt  time.Time
}

var (
	db          *sql.DB
	once        sync.Once
	mu          sync.Mutex
	CurrentRate float64
)

func init() {
	err := os.MkdirAll(config.CurrentConfig.Data, os.ModePerm)
	if err != nil {
		log.Fatal("Faild to make directiry: ", err)
	}
	GetRate()
	runCron()
}

// InitDB initializes the SQLite file and creates the usage_records table
func InitDB(dbPath string) {
	once.Do(func() {
		var err error
		db, err = sql.Open("sqlite3", dbPath)
		if err != nil {
			log.Fatal("SQLiteオープンエラー: ", err)
		}

		// create table if not exists
		createTableSQL := `
		CREATE TABLE IF NOT EXISTS usage_records (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			server_id     TEXT    NOT NULL,
			model         TEXT    NOT NULL,
			usage_type    TEXT    NOT NULL,
			usage_count   INTEGER NOT NULL,
			month         INTEGER NOT NULL,
			year          INTEGER NOT NULL,
			created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(server_id, model, usage_type, month, year)
		);
		`
		_, err = db.Exec(createTableSQL)
		if err != nil {
			log.Fatal("テーブル作成失敗: ", err)
		}
	})
}

func AddUsage(serverID, model, usageType string, addCount int) error {
	if db == nil {
		return fmt.Errorf("database is not initialized. Please call InitDB first")
	}

	mu.Lock()
	defer mu.Unlock()

	if addCount == 0 {
		return nil
	}
	now := time.Now()
	month := int(now.Month())
	year := now.Year()

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// find the current record (server_id, model, usage_type, month, year)
	querySelect := `
		SELECT usage_count FROM usage_records
		WHERE server_id = ? AND model = ? AND usage_type = ? AND month = ? AND year = ?
	`
	var currentCount int
	err = tx.QueryRow(querySelect, serverID, model, usageType, month, year).Scan(&currentCount)

	if err == sql.ErrNoRows {
		// If not exit records, INSERT
		insertSQL := `
			INSERT INTO usage_records (server_id, model, usage_type, usage_count, month, year)
			VALUES (?, ?, ?, ?, ?, ?)
		`
		_, err = tx.Exec(insertSQL, serverID, model, usageType, addCount, month, year)
		if err != nil {
			tx.Rollback()
			return err
		}
	} else if err != nil {
		tx.Rollback()
		return err
	} else {
		// if exit records, UPDATE
		newCount := currentCount + addCount
		updateSQL := `
			UPDATE usage_records
			SET usage_count = ?
			WHERE server_id = ? AND model = ? AND usage_type = ? AND month = ? AND year = ?
		`
		_, err = tx.Exec(updateSQL, newCount, serverID, model, usageType, month, year)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func GetMonthlyUsage(serverID string) ([]UsageRecord, error) {
	now := time.Now()
	month := int(now.Month())
	year := now.Year()

	query := `
		SELECT id, server_id, model, usage_type, usage_count, month, year, created_at
		FROM usage_records
		WHERE server_id = ? AND month = ? AND year = ?
	`
	rows, err := db.Query(query, serverID, month, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []UsageRecord
	for rows.Next() {
		var r UsageRecord
		err := rows.Scan(
			&r.ID,
			&r.ServerID,
			&r.Model,
			&r.UsageType,
			&r.UsageCount,
			&r.Month,
			&r.Year,
			&r.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	return results, nil
}

func CalcCost(records []UsageRecord) float64 {
	var totalUSD float64

	for _, rec := range records {
		// Check if the model is registered in the config
		modelInfo, ok := config.AllModels[rec.Model]
		if !ok {
			continue
		}

		usageType := rec.UsageType
		usageCount := float64(rec.UsageCount)

		if modelInfo.Type == config.ModelTypeText {
			// calculate the cost(based on 1M tokens)
			switch usageType {
			case "prompt_tokens":
				totalUSD += (usageCount / 1000000.0) * modelInfo.PromptCost
			case "completion_tokens":
				totalUSD += (usageCount / 1000000.0) * modelInfo.CompletionCost
			case "prompt_cache_tokens":
				// Cache tokens are half the price of prompt tokens
				totalUSD += (usageCount / 1000000.0) * (modelInfo.PromptCost / 2.0)
			case "vision_tokens":
				totalUSD += (usageCount / 1000000.0) * modelInfo.VisionCost.Fixed
			}

		} else if modelInfo.Type == config.ModelTypeImage {
			//Get the prefix and determine the model variant
			variant := splitImageModelInfo(usageType)
			variant_suffix := variant[len(variant)-1]

			// dall-e-2
			if variant_suffix == "small" || variant_suffix == "medium" || variant_suffix == "big" {
				switch variant_suffix {
				case "small":
					totalUSD += modelInfo.ImageCost["small"] * usageCount
				case "medium":
					totalUSD += modelInfo.ImageCost["medium"] * usageCount
				case "big":
					totalUSD += modelInfo.ImageCost["big"] * usageCount
				}
			}

			// dall-e-3
			if variant_suffix == "square" || variant_suffix == "rectangle" {
				// For dall-e-3, since "square" etc. remain, determine if "hd" is included.
				if variant[3] == "standard" {
					switch variant_suffix {
					case "square":
						totalUSD += modelInfo.ImageCost["standard-square"] * usageCount
					case "rectangle":
						totalUSD += modelInfo.ImageCost["standard-rectangle"] * usageCount
					}
				} else if variant[3] == "hd" {
					switch variant_suffix {
					case "square":
						totalUSD += modelInfo.ImageCost["hd-square"] * usageCount
					case "rectangle":
						totalUSD += modelInfo.ImageCost["hd-rectangle"] * usageCount
					}
				}
			}
		}
	}

	return totalUSD
}

func splitImageModelInfo(modelInfo string) []string {
	split := strings.Split(modelInfo, "-")
	return split
}

func GetRate() {
	CurrentRate = 145

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	url := "https://api.excelapi.org/currency/rate?pair=usd-jpy"

	resp, err := client.Get(url)
	if err != nil {
		if err, ok := err.(net.Error); ok && err.Timeout() {
			log.Println("API timeout: ", err)
		}
		log.Print("API for get rate error: ", err)
		return
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return
	}

	byteArray, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Print("Reading body error: ", err)
		return
	}

	CurrentRate, err = strconv.ParseFloat(string(byteArray), 64)
	if err != nil {
		log.Print("Parsing rate error: ", err)
		return
	}
}

func runCron() {
	c := cron.New()
	c.AddFunc("0 0 * * *", func() { GetRate() })
	c.Start()
}
