package monitor

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v2"
)

type MintConfig struct {
	Blacklist []string `yaml:"blacklist"`
}

type RobotConfig struct {
	Robot []string `yaml:"robot"`
}

type SmartConfig struct {
	Addresses map[string]string `yaml:"addresses"`
}

type StrategyParams struct {
	MaxBuyAmount       float64 `yaml:"max_buy_amount"`
	MinBuyAmount       float64 `yaml:"min_buy_amount"`
	MaxHoldMillisecond int     `yaml:"max_hold_millisecond"`
	MinHoldMillisecond int     `yaml:"min_hold_millisecond"`
	BuySlippage        float64 `yaml:"buy_slippage"`
	SellSlippage       float64 `yaml:"sell_slippage"`
	DelayMillisecond   int     `yaml:"delay_millisecond"`
	MintStart          bool    `yaml:"mint_start"`
	SmartStart         bool    `yaml:"smart_start"`
}

type StrategyConfig struct {
	Default StrategyParams            `yaml:"default"`
	Hourly  map[string]StrategyParams `yaml:"hourly"`
}

var (
	strategyConfig *StrategyConfig
	mintConfig     *MintConfig
	robotConfig    *RobotConfig
	configLock     sync.RWMutex
)

var (
	smartConfig    *SmartConfig
	configMu       sync.RWMutex
	lastModTime    time.Time
	configFilePath = "config/smart_addresses.yaml"
)

// 初始化配置（启动时调用）
func InitConfig() {
	ReloadConfig()
	ReloadSmartConfig()

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		for range ticker.C {
			if err := ReloadConfig(); err != nil {
				log.Println("reload config failed:", err)
			} else {
				log.Println("strategy config reloaded")
			}
		}
	}()
}

// 热重载配置
func ReloadSmartConfig() error {
	file, err := os.Open(configFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	// 检查文件是否修改
	if !info.ModTime().After(lastModTime) {
		return nil
	}

	var newConfig SmartConfig
	if err := yaml.NewDecoder(file).Decode(&newConfig); err != nil {
		return err
	}

	configMu.Lock()
	defer configMu.Unlock()
	smartConfig = &newConfig
	lastModTime = info.ModTime()
	return nil
}

func ReloadConfig() error {
	log.Println("reload config")
	configLock.Lock()
	defer configLock.Unlock()

	var strategyCfg StrategyConfig
	if err := LoadYAMLConfig("config/strategy.yaml", &strategyCfg); err != nil {
		log.Fatalf("加载策略配置失败: %v", err)
		return err
	}
	strategyConfig = &strategyCfg

	var mintCfg MintConfig
	if err := LoadYAMLConfig("config/mint.yaml", &mintCfg); err != nil {
		log.Fatalf("加载Mint配置失败: %v", err)
		return err
	}
	mintConfig = &mintCfg

	var robotCfg RobotConfig
	if err := LoadYAMLConfig("config/robot.yaml", &robotCfg); err != nil {
		log.Fatalf("加载机器人配置失败: %v", err)
		return err
	}
	robotConfig = &robotCfg

	return nil
}

// 获取当前配置（线程安全）
func GetSmartAddresses() map[string]string {
	configMu.RLock()
	defer configMu.RUnlock()
	if smartConfig == nil {
		return make(map[string]string)
	}
	return smartConfig.Addresses
}

func IsSmartAddress(address string) bool {
	config := GetSmartAddresses()
	_, ok := config[address]
	return ok
}

func WatchConfigChanges(restartChan chan<- struct{}) {
	absPath, err := filepath.Abs(configFilePath)
	if err != nil {
		log.Fatalf("解析配置文件绝对路径失败: %v", err)
	}

	dirPath := filepath.Dir(absPath)
	fileName := filepath.Base(absPath)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("创建文件监听器失败: %v", err)
	}
	defer watcher.Close()

	err = watcher.Add(dirPath)
	if err != nil {
		log.Fatalf("添加监听目录失败: %v", err)
	}

	log.Println("开始监听配置文件变更:", absPath)

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// 只处理目标配置文件的事件
			if filepath.Base(event.Name) != fileName {
				continue
			}

			// 只处理 Write / Rename / Create
			if event.Op&(fsnotify.Write|fsnotify.Rename|fsnotify.Create) != 0 {
				log.Printf("检测到配置文件变更: %s, 操作: %s", event.Name, event.Op.String())

				// 检查文件是否存在（避免 Rename 后一时未落盘）
				time.Sleep(200 * time.Millisecond)
				if _, err := os.Stat(absPath); os.IsNotExist(err) {
					log.Printf("变更后配置文件不存在，跳过处理")
					continue
				}

				// 热加载配置
				if err := ReloadConfig(); err != nil {
					log.Printf("配置热更新失败: %v", err)
				}

				restartChan <- struct{}{}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("监听错误: %v", err)
		}
	}
}

func GetMintConfig() (*MintConfig, error) {
	configLock.Lock()
	defer configLock.Unlock()
	if mintConfig != nil {
		return mintConfig, nil
	}
	return nil, errors.New("未加载配置文件")
}

func GetStrategyConfig() (*StrategyConfig, error) {
	configLock.Lock()
	defer configLock.Unlock()
	if strategyConfig != nil {
		return strategyConfig, nil
	}
	return nil, errors.New("未加载配置文件")
}

func GetRobotConfig() (*RobotConfig, error) {
	configLock.Lock()
	defer configLock.Unlock()
	if robotConfig != nil {
		return robotConfig, nil
	}
	return nil, errors.New("未加载配置文件")
}

func LoadYAMLConfig(path string, cfg any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, cfg)
}

func GetStrategyParamsByHour(hour int) StrategyParams {
	configLock.RLock()
	defer configLock.RUnlock()

	defaults := strategyConfig.Default
	override, ok := strategyConfig.Hourly[strconv.Itoa(hour)]
	if !ok {
		return defaults
	}

	// 合并 default 和 override
	result := defaults
	if override.MaxBuyAmount != 0 {
		result.MaxBuyAmount = override.MaxBuyAmount
	}
	if override.MinBuyAmount != 0 {
		result.MinBuyAmount = override.MinBuyAmount
	}
	if override.MaxHoldMillisecond != 0 {
		result.MaxHoldMillisecond = override.MaxHoldMillisecond
	}
	if override.MinHoldMillisecond != 0 {
		result.MinHoldMillisecond = override.MinHoldMillisecond
	}
	if override.BuySlippage != 0 {
		result.BuySlippage = override.BuySlippage
	}
	if override.SellSlippage != 0 {
		result.SellSlippage = override.SellSlippage
	}
	if override.DelayMillisecond != 0 {
		result.DelayMillisecond = override.DelayMillisecond
	}
	if override.MintStart {
		result.MintStart = override.MintStart
	}
	if override.SmartStart {
		result.SmartStart = override.SmartStart
	}
	return result
}
