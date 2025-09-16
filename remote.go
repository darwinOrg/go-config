package dgcfg

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
	_ "github.com/spf13/viper/remote"
)

// RemoteProvider 远程配置提供者结构体
type RemoteProvider struct {
	Provider   string // 提供者类型：consul、etcd、nacos、apollo等
	Endpoint   string // 提供者地址
	Path       string // 配置在提供者中的路径
	ConfigType string // 配置格式：json、yaml等
	Keyring    string // 用于加密配置的密钥(可选)
}

// RemoteConfig 远程配置结构体
type RemoteConfig struct {
	Providers     []RemoteProvider // 远程配置提供者列表
	Watch         bool             // 是否监听配置变化
	WatchInterval time.Duration    // 监听间隔
}

// ReadRemoteConfig 从远程配置中心读取配置
func ReadRemoteConfig[T any](remoteConfig *RemoteConfig) (*T, error) {
	if len(remoteConfig.Providers) == 0 {
		return nil, fmt.Errorf("no remote providers configured")
	}

	// 使用第一个配置提供者
	provider := remoteConfig.Providers[0]

	viper.SetConfigType(provider.ConfigType)
	err := viper.AddRemoteProvider(provider.Provider, provider.Endpoint, provider.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to add remote provider: %v", err)
	}

	// 读取远程配置
	if err := viper.ReadRemoteConfig(); err != nil {
		return nil, fmt.Errorf("failed to read remote config: %v", err)
	}

	// 如果启用了监听，则启动一个goroutine来监听配置变化
	if remoteConfig.Watch {
		go func() {
			for {
				time.Sleep(remoteConfig.WatchInterval)
				err := viper.WatchRemoteConfig()
				if err != nil {
					log.Printf("failed to watch remote config: %v", err)
					continue
				}
				log.Println("remote config updated")
			}
		}()
	}

	c := new(T)
	err = viper.Unmarshal(c)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal remote config: %v", err)
	}

	return c, nil
}

// ParseRemoteConfig 从环境变量解析远程配置
func ParseRemoteConfig() *RemoteConfig {
	providersEnv := os.Getenv("CONFIG_REMOTE_PROVIDERS")
	if providersEnv == "" {
		return nil
	}

	remoteConfig := &RemoteConfig{
		Providers: []RemoteProvider{},
		Watch:     os.Getenv("CONFIG_REMOTE_WATCH") == "true",
	}

	watchInterval := os.Getenv("CONFIG_REMOTE_WATCH_INTERVAL")
	if watchInterval != "" {
		if interval, err := time.ParseDuration(watchInterval); err == nil {
			remoteConfig.WatchInterval = interval
		} else {
			remoteConfig.WatchInterval = time.Second * 30 // 默认30秒
		}
	} else {
		remoteConfig.WatchInterval = time.Second * 30 // 默认30秒
	}

	// 解析远程提供者配置
	// 格式: provider1|endpoint1|path1|configType1|keyring1;provider2|endpoint2|path2|configType2|keyring2
	providerStrings := strings.Split(providersEnv, ";")
	for _, providerStr := range providerStrings {
		parts := strings.Split(providerStr, "|")
		if len(parts) >= 4 {
			provider := RemoteProvider{
				Provider:   parts[0],
				Endpoint:   parts[1],
				Path:       parts[2],
				ConfigType: parts[3],
			}
			if len(parts) >= 5 {
				provider.Keyring = parts[4]
			}
			remoteConfig.Providers = append(remoteConfig.Providers, provider)
		}
	}

	if len(remoteConfig.Providers) == 0 {
		return nil
	}

	return remoteConfig
}
