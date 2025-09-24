package models

type Config struct {
	APISites map[string]APISite `json:"api_site"`
}

type APISite struct {
	Key    string `json:"key"`
	API    string `json:"api"`
	Name   string `json:"name"`
	Detail string `json:"detail"`
}
