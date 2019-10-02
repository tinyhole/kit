package web

import (
	"flag"
	"fmt"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/registry/consul"
	"github.com/micro/go-micro/web"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"time"
)

var (
	GitTag    = "2000.01.01.release"
	BuildTime = "2000-01-01T00:00:00+0800"
)

var (
	DefaultService web.Service
)

func Init() {
	//显示版本号信息
	version := flag.Bool("v", false, "version")
	flag.Parse()

	if *version {
		fmt.Println("Git Tag: " + GitTag)
		fmt.Println("Build Time: " + BuildTime)
		os.Exit(0)
	}

	LoadConfig()

	DefaultService = web.NewService(
		web.Address(DefaultServiceConf.Address),
		web.Name(DefaultServiceConf.Name),
		web.RegisterTTL(time.Second*30),
		web.RegisterInterval(time.Second*10),
		web.Version(DefaultServiceConf.Version),
		web.Metadata(DefaultServiceConf.Metadata),
		web.Registry(consul.NewRegistry(registry.Addrs(DefaultRegistryConf.Address))),
	)

	DefaultService.Init()

}

func Run() {
	if err := DefaultService.Run(); err != nil {
		fmt.Println("err:", err)
		logrus.Fatalf("service run error: %v", err)
	}
}

func ServiceName() string {
	return DefaultServiceConf.Name
}

func ServiceVersion() string {
	return DefaultServiceConf.Version
}

func ServiceMetadata(key, def string) string {
	val, ok := DefaultServiceConf.Metadata[key]
	if ok {
		return val
	}
	return def
}

func Client() *http.Client {
	return DefaultService.Client()
}

