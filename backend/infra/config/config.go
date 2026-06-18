package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config 聚合所有运行时配置。
type Config struct {
	HTTPAddr          string
	MySQLDSN          string
	PythonLLMEndpoint string
}

// Load 从 config.yaml 读取配置，并允许环境变量覆盖（APP_ 前缀）。
// 加载顺序：YAML 默认值 → 环境变量覆盖
func Load() Config {
	v := viper.New()

	// --- YAML 配置 ---
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")         // 运行时当前目录（go run 场景）
	v.AddConfigPath("..")        // 从子目录运行时
	v.AddConfigPath("./backend") // 从项目根目录运行时

	if err := v.ReadInConfig(); err != nil {
		// 配置文件不存在时仅打印警告，使用环境变量 + 零值回退
		fmt.Printf("[config] 警告: 未找到 config.yaml (%v)，将仅使用环境变量\n", err)
	}

	// --- 环境变量覆盖（APP_ 前缀） ---
	v.SetEnvPrefix("APP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// --- 绑定到 Config 结构体 ---
	return Config{
		HTTPAddr:          v.GetString("server.http_addr"),
		MySQLDSN:          v.GetString("database.mysql_dsn"),
		PythonLLMEndpoint: v.GetString("llm.python_endpoint"),
	}
}
