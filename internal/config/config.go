package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/danger-dream/ebpf-firewall/internal/types"
	"github.com/danger-dream/ebpf-firewall/internal/utils"

	"gopkg.in/yaml.v2"
)

type ConfigManager struct {
	configPath   string
	Config       *types.Config
	blackChannel chan types.WebSocketChangeBlackListPayload
}

func NewConfigManager(path string) (*ConfigManager, error) {
	cm := &ConfigManager{
		configPath:   path,
		blackChannel: make(chan types.WebSocketChangeBlackListPayload, 1),
	}
	if err := cm.LoadConfig(); err != nil {
		return nil, err
	}
	return cm, nil
}

func (cm *ConfigManager) LoadConfig() error {
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return fmt.Errorf("error reading config file: %v", err)
	}

	var config types.Config
	ext := filepath.Ext(cm.configPath)
	if ext == ".json" {
		err = json.Unmarshal(data, &config)
	} else if ext == ".yaml" || ext == ".yml" {
		err = yaml.Unmarshal(data, &config)
	} else {
		return fmt.Errorf("unsupported config file format: %s", ext)
	}

	if err != nil {
		return fmt.Errorf("error parsing config file: %v", err)
	}
	if config.Interface == "" {
		config.Interface = utils.GetDefaultInterface()
	}
	if !utils.CheckAddr(config.Addr) {
		return fmt.Errorf("addr 格式错误")
	}
	cm.Config = &config
	return nil
}

func (cm *ConfigManager) SaveConfig() error {
	ext := filepath.Ext(cm.configPath)
	var data []byte
	var err error
	if ext == ".json" {
		data, err = json.MarshalIndent(cm.Config, "", "\t")
		if err != nil {
			return fmt.Errorf("error marshalling config: %v", err)
		}
	} else if ext == ".yaml" || ext == ".yml" {
		data, err = yaml.Marshal(cm.Config)
		if err != nil {
			return fmt.Errorf("error marshalling config: %v", err)
		}
	} else {
		return fmt.Errorf("unsupported config file format: %s", ext)
	}
	return os.WriteFile(cm.configPath, data, 0644)
}

func (cm *ConfigManager) UpdateBlackList(black types.WebSocketChangeBlackListPayload) error {
	isValid := false
	switch black.Type {
	case "mac":
		isValid = utils.IsValidMAC(black.Data)
	case "ipv4":
		isValid = utils.IsValidIPv4(black.Data)
	case "ipv6":
		isValid = utils.IsValidIPv6(black.Data)
	}
	if !isValid {
		return fmt.Errorf("%s 地址校验失败", black.Type)
	}
	if black.Inc {
		if black.Type == "mac" {
			cm.Config.Black.Mac = append(cm.Config.Black.Mac, black.Data)
		} else if black.Type == "ipv4" {
			cm.Config.Black.Ipv4 = append(cm.Config.Black.Ipv4, black.Data)
		} else if black.Type == "ipv6" {
			cm.Config.Black.Ipv6 = append(cm.Config.Black.Ipv6, black.Data)
		}
	} else {
		if black.Type == "mac" {
			cm.Config.Black.Mac = utils.RemoveStringFromSlice(cm.Config.Black.Mac, black.Data)
		} else if black.Type == "ipv4" {
			cm.Config.Black.Ipv4 = utils.RemoveStringFromSlice(cm.Config.Black.Ipv4, black.Data)
		} else if black.Type == "ipv6" {
			cm.Config.Black.Ipv6 = utils.RemoveStringFromSlice(cm.Config.Black.Ipv6, black.Data)
		}
	}
	if err := cm.SaveConfig(); err != nil {
		return err
	}
	cm.blackChannel <- black
	return nil
}

func (cm *ConfigManager) GetBlackChannel() chan types.WebSocketChangeBlackListPayload {
	return cm.blackChannel
}
