package web


import (
	"fmt"
	"github.com/tinyhole/kit/log"
	"github.com/micro/go-micro/config"
	"github.com/micro/go-micro/config/encoder/yaml"
	"github.com/micro/go-micro/config/source"
	"github.com/micro/go-micro/config/source/consul"
	"github.com/micro/go-micro/config/source/file"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"strings"
)

type (
	RegistryConfig struct {
		Address string `json:"address"`
	}

	ServiceConfig struct {
		Address string `json:"address"`
		Name         string `json:"name"`
		Version      string `json:"version"`
		Metadata     map[string]string `json:"metadata"`
	}

	LogConfig struct {
		Path     string `json:"path"`
		Level    string `json:"level"`
		FileSize int `json:"fileSize"`
	}
)

var (
	DefaultRegistryConf RegistryConfig
	DefaultServiceConf ServiceConfig //服务配置
	DefaultLogConf LogConfig
)

func LoadConfig() {
	// 加载最基础的配置
	err := config.Load(file.NewSource(file.WithPath("config.yaml")))
	if err != nil {
		// load from consul
		fmt.Println("load config from consul")
		addr := os.Getenv("K8S_SERVER_CONFIG_ADDR")
		path := os.Getenv("K8S_SERVER_CONFIG_PATH")
		fmt.Println("addr:", addr)
		fmt.Println("path:", path)
		err = config.Load(consul.NewSource(
			consul.WithAddress(addr),
			consul.WithPrefix(path),
			consul.StripPrefix(true),
			source.WithEncoder(yaml.NewEncoder())))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
	// 加载注册中心地址
	err = config.Get("registry").Scan(&DefaultRegistryConf)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	//检测是否是环境变量
	if strings.Index(DefaultRegistryConf.Address, "$") == 0 {
		DefaultRegistryConf.Address = os.Getenv(strings.TrimPrefix(DefaultRegistryConf.Address, "$"))
	}

	// 加载服务配置
	err = config.Get("service").Scan(&DefaultServiceConf)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// 加载并初始化日志配置
	loadLogConfAndInitLogger()

	fmt.Println("registry:", DefaultRegistryConf)
	fmt.Println("service:", DefaultServiceConf)
	fmt.Println("log:", DefaultLogConf)
}

func loadLogConfAndInitLogger() {
	err := config.Get("log").Scan(&DefaultLogConf)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	level, _ := logrus.ParseLevel(DefaultLogConf.Level)
	logrus.SetLevel(level)
	fileIO := log.NewLogFile(
		log.FilePath(DefaultLogConf.Path),
		log.FileSize(DefaultLogConf.FileSize),
		log.FileTime(true))
	logIO := io.MultiWriter(fileIO,os.Stdout)
	logrus.SetOutput(logIO)

	go watchLogConf()
}

func watchLogConf() {
	w, err := config.Watch("log")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for {
		v, err := w.Next()
		if err != nil {
			fmt.Println(err)
			continue
		}

		err = v.Scan(&DefaultLogConf)
		if err != nil {
			continue
		}

		level, _ := logrus.ParseLevel(DefaultLogConf.Level)
		logrus.SetLevel(level)

		logrus.SetOutput(log.NewLogFile(
			log.FilePath(DefaultLogConf.Path),
			log.FileSize(DefaultLogConf.FileSize),
			log.FileTime(true)))
	}

}


