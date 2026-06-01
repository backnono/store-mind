package config

import "os"

type Config struct {
	HTTPAddr string
	MySQLDSN string
}

func Load() Config {
	cfg := Config{
		HTTPAddr: ":8080",
		MySQLDSN: "app:app@tcp(127.0.0.1:3307)/store_mind?parseTime=true",
	}
	if v := os.Getenv("HTTP_ADDR"); v != "" {
		cfg.HTTPAddr = v
	}
	if v := os.Getenv("MYSQL_DSN"); v != "" {
		cfg.MySQLDSN = v
	}
	return cfg
}
