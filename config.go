package dgcfg

import (
	"fmt"
	dgsys "github.com/darwinOrg/go-common/sys"
	"github.com/darwinOrg/go-common/utils"
	"github.com/jinzhu/copier"
	"github.com/spf13/viper"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
)

var (
	ConfigName = "app"
	ConfigType = "yml"
)

func ReadConfigDefault[T any]() (*T, error) {
	confRoot := MustConfRoot()

	profile := dgsys.GetProfile()
	if profile == "" {
		return ReadConfigFile[T](confRoot, ConfigName, ConfigType)
	}

	profileConfigName := fmt.Sprintf("app-%s", profile)
	if !utils.ExistsFile(filepath.Join(confRoot, profileConfigName+"."+ConfigType)) {
		return ReadConfigFile[T](confRoot, ConfigName, ConfigType)
	}

	if !utils.ExistsFile(filepath.Join(confRoot, ConfigName+"."+ConfigType)) {
		return ReadConfigFile[T](confRoot, profileConfigName, ConfigType)
	}

	cfg1, err := ReadConfigFile[T](confRoot, ConfigName, ConfigType)
	if err != nil {
		return nil, err
	}

	cfg2, err := ReadConfigFile[T](confRoot, profileConfigName, ConfigType)
	if err != nil {
		return nil, err
	}

	err = copier.CopyWithOption(cfg1, cfg2, copier.Option{IgnoreEmpty: true, DeepCopy: true})
	if err != nil {
		log.Printf("copier.Copy error: %v", err)
		return nil, err
	}

	return cfg1, nil
}

func ReadConfigFile[T any](confRoot string, configName string, configType string) (*T, error) {
	log.Printf("use confRoot: %s, configName: %s", confRoot, configName)
	viper.SetConfigFile(filepath.Join(confRoot, configName+"."+configType))
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	viper.AutomaticEnv()
	configContent := viper.AllSettings()
	replacedConfig := replaceEnvVariables(configContent)

	if err := viper.MergeConfigMap(replacedConfig); err != nil {
		return nil, err
	}

	c := new(T)
	err := viper.Unmarshal(c)
	if err != nil {
		log.Printf("viper.Unmarshal error | confRoot: %s | configName: %s | err: %v", confRoot, configName, err)
	}

	return c, err
}

func MustConfRoot() string {
	confRoot := os.Getenv("CONF_ROOT")
	if confRoot == "" {
		confRoot = "resources"
	}

	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to get current working directory: %v", err)
	}

	for {
		if utils.ExistsDir(path.Join(currentDir, confRoot)) {
			return path.Join(currentDir, confRoot)
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break
		}
		currentDir = parentDir
	}

	log.Fatal("can not find confRoot")
	return ""
}

// 替换配置中的环境变量占位符
func replaceEnvVariables(config map[string]any) map[string]any {
	for key, value := range config {
		switch value.(type) {
		case string:
			config[key] = replaceStringEnv(value.(string))
		case map[string]any:
			config[key] = replaceEnvVariables(value.(map[string]any))
		}
	}
	return config
}

// 替换字符串中的环境变量占位符，支持默认值
func replaceStringEnv(value string) string {
	re := regexp.MustCompile(`\${(.*?)(?::(.*?))?}`)
	return re.ReplaceAllStringFunc(value, func(match string) string {
		// 提取环境变量名和默认值
		parts := re.FindStringSubmatch(match)
		envName := parts[1]
		defaultValue := parts[2]

		// 获取环境变量值
		envValue := os.Getenv(envName)
		if envValue == "" && defaultValue != "" {
			return defaultValue
		}
		return envValue
	})
}
