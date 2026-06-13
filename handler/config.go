package handler

import (
	"encoding/json"
	"fmt"
)

type Config struct {
	Build   string   `json:"build"`
	Sources []string `json:"sources"`

	RegexpError   string `json:"re_error"`
	RegexpSources string `json:"re_sources"`
}

func loadConfig(options any) (Config, error) {
	b, err := json.Marshal(options)
	if err != nil {
		return Config{}, fmt.Errorf("failed to serialize initialization options: %w", err)
	}

	var config Config

	if err := json.Unmarshal(b, &config); err != nil {
		return Config{}, fmt.Errorf("failed to parse initialization options: %w", err)
	}
	return config, nil
}
