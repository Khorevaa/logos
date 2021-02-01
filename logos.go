package logos

import (
	"errors"
	"fmt"
	"github.com/khorevaa/logos/config"
	"github.com/khorevaa/logos/internal/common"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"

	_ "github.com/khorevaa/logos/appender"
	_ "github.com/khorevaa/logos/encoder/common"
	_ "github.com/khorevaa/logos/encoder/console"
	_ "github.com/khorevaa/logos/encoder/gelf"
	_ "github.com/khorevaa/logos/encoder/json"
)

var (
	manager        *logManager
	configFile     string
	initLocker     sync.Mutex
	explicitInited = false
	debug          bool
)

func resolveConfigFileFromEnv() (string, error) {
	f := os.Getenv("LOGOS_CONFIG_FILE")
	if f == "" {
		return "", errors.New("environment variable 'LOGOS_CONFIG_FILE' is not set")
	}
	return f, nil
}

func resolveConfigFileFromWorkDir() (string, error) {
	matches1, _ := filepath.Glob("logos.yaml")
	matches2, _ := filepath.Glob("logos.yml")
	matches := append(matches1, matches2...)
	switch len(matches) {
	case 0:
		return "", errors.New("no config file found in work dir")
	case 1:
		return matches[0], nil
	default:
		panic(fmt.Errorf("multiple config files found %v", matches))
	}
}

func init() {

	initLocker.Lock()
	defer initLocker.Unlock()

	debug, _ = strconv.ParseBool(os.Getenv("LOGOS_DEBUG"))

	if configFile == "" {
		cf, err := resolveConfigFileFromEnv()
		if err == nil {
			configFile = cf
		}
	}

	if configFile == "" {
		cf, err := resolveConfigFileFromWorkDir()
		if err == nil {
			configFile = cf
		}
	}

	var err error
	var rawConfig *common.Config

	if configFile != "" {
		// load ConfigFile
		configFile, err = filepath.Abs(configFile)
		if err != nil {
			panic(err)
		}

		if debug {
			fmt.Println("logos using config file: ", configFile)
			bs, err := ioutil.ReadFile(configFile)
			if err != nil {
				panic(err)
			}
			fmt.Println(string(bs))
		}

		rawConfig, _, err = common.LoadFile(configFile)
	} else {
		if debug {
			fmt.Print("logos using default config:\n" + config.DefaultConfig)
		}
		rawConfig, err = common.NewConfigFrom(config.DefaultConfig)
	}

	if err != nil {
		panic(err)
	}

	envConfig, err := parseConfigFromEnv()
	if err != nil && debug {
		fmt.Printf("logos loading config from env err: %s", err)
	}
	if envConfig != nil {
		rawConfig, err = common.MergeConfigs(rawConfig, envConfig)
		if err != nil {
			fmt.Printf("logos merge configs err: %s", err)
		}
	}

	manager, err = newLogManager(rawConfig)

	if err != nil {
		panic(err)
	}

	manager.RedirectStdLog()

	go func() {
		quit := make(chan os.Signal)
		signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
		<-quit
		Sync()
	}()

}
func parseConfigFromEnv() (*common.Config, error) {
	configData := os.Getenv("LOGOS_CONFIG")
	if configData == "" {
		return nil, ErrEnvConfigNotSet
	}
	return parseConfigFromString(configData)
}

func parseConfigFromString(configData string) (*common.Config, error) {

	newConfig := common.NewConfig()

	data := strings.Split(configData, ";")

	for _, strData := range data {

		pathValue := strings.Split(strData, "=")
		value := ""
		path := pathValue[0]
		if len(pathValue) == 2 {
			value = pathValue[1]
		}
		err := newConfig.SetString(path, -1, value)
		if debug {
			fmt.Errorf("error loading config from path %s err <%s>", path, err)
		}

	}

	return newConfig, nil
}

func InitWithConfigContent(content string) error {
	initLocker.Lock()
	defer initLocker.Unlock()

	//if explicitInited {
	//	return errors.New("logos is explicit inited")
	//}

	if debug {
		fmt.Println("logos InitWithConfigContent:\n" + content)
	}

	rawConfig, err := common.NewConfigFrom(content)
	if err != nil {
		return err
	}

	err = manager.Update(rawConfig)
	if err != nil {
		return err
	}

	//explicitInited = true

	return nil
}

func New(name string) Logger {
	manager.Sync()
	return manager.NewLogger(name)
}
func SetLevel(name string, level zapcore.Level) {
	manager.SetLevel(name, level)
}

func UpdateLogger(name string, logger *zap.Logger) {
	manager.UpdateLogger(name, logger)
}

func Sync() {
	_ = manager.Sync()
}

func RedirectStdLog() func() {
	return manager.RedirectStdLog()
}

func CancelRedirectStdLog() {
	manager.CancelRedirectStdLog()
}
