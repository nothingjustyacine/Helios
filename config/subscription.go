package config

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"helios/models"

	"github.com/btcsuite/btcutil/base58"
	"github.com/bytedance/sonic"
)

var GlobalConfig models.Config

func FetchSubscription(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch subscription: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("subscription request failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	decodedData := base58.Decode(string(body))
	if len(decodedData) == 0 {
		return fmt.Errorf("failed to decode base58 data")
	}

	var config models.Config
	if err := sonic.UnmarshalString(string(decodedData), &config); err != nil {
		return fmt.Errorf("failed to unmarshal config JSON: %v", err)
	}
	if len(config.APISites) == 0 {
		return fmt.Errorf("no API sites found in subscription")
	}
	for k, v := range config.APISites {
		v.Key = k
		config.APISites[k] = v
	}

	GlobalConfig = config
	log.Printf("Subscription config loaded successfully. API sites: %d", len(config.APISites))
	return nil
}
