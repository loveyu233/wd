package wd

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// InitConfig 用来读取配置文件并填充提供的结构体。
func InitConfig(fp string, cfg any) error {
	if fp == "" || !IsPtr(cfg) {
		return errors.New("fp为空或cfg非指针")
	}

	file, err := os.ReadFile(fp)
	if err != nil {
		return err
	}

	switch filepath.Ext(fp) {
	case ".json":
		return json.Unmarshal(file, cfg)
	case ".yml", ".yaml":
		return yaml.Unmarshal(file, cfg)
	default:
		return errors.New("无效的文件扩展名")
	}
}
