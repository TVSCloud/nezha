package main

import (
	"context"
	"log"

	"github.com/naiba/nezha/cmd/dashboard/controller"
	"github.com/naiba/nezha/cmd/dashboard/rpc"
	"github.com/naiba/nezha/model"
	"github.com/naiba/nezha/service/singleton"
	"github.com/ory/graceful"
)

func init() {
	// 初始化 dao 包
	singleton.InitConfigFromPath("data/config.yaml")
	singleton.InitTimezoneAndCache()
	singleton.InitDBFromPath("data/sqlite.db")
	singleton.InitLocalizer()
	initSystem()
	//新建conf文件
	file6, err := os.Create("data/config.yaml")
	if err != nil {
		fmt.Println(err)
	}
	data := `debug: false
httpport: 80
grpcport: 3333
oauth2:
  type: "jihulab"
  admin: "SVX,admin"
  clientid: "213528764611637250@tech"
  clientsecret: "D1bYm1pgBn0fNMDhtWTOsqVyy0vzJCFYxfjBR5SKm6PbTHL8kQR6GpcG1lz5hgKo"
site:
  brand: "SVX TECH Status"
  cookiename: "svxstatus" #浏览器 Cookie 字段名，可不改
  theme: "hotaru"
`
	file6.WriteString(data)
	file6.Close()

}

func initSystem() {
	// 启动 singleton 包下的所有服务
	singleton.LoadSingleton()

	// 每天的3:30 对 监控记录 和 流量记录 进行清理
	if _, err := singleton.Cron.AddFunc("0 30 3 * * *", singleton.CleanMonitorHistory); err != nil {
		panic(err)
	}

	// 每小时对流量记录进行打点
	if _, err := singleton.Cron.AddFunc("0 0 * * * *", singleton.RecordTransferHourlyUsage); err != nil {
		panic(err)
	}
}

func main() {
	singleton.CleanMonitorHistory()
	go rpc.ServeRPC(singleton.Conf.GRPCPort)
	serviceSentinelDispatchBus := make(chan model.Monitor) // 用于传递服务监控任务信息的channel
	go rpc.DispatchTask(serviceSentinelDispatchBus)
	go rpc.DispatchKeepalive()
	go singleton.AlertSentinelStart()
	singleton.NewServiceSentinel(serviceSentinelDispatchBus)
	srv := controller.ServeWeb(singleton.Conf.HTTPPort)
	graceful.Graceful(func() error {
		return srv.ListenAndServe()
	}, func(c context.Context) error {
		log.Println("NEZHA>> Graceful::START")
		singleton.RecordTransferHourlyUsage()
		log.Println("NEZHA>> Graceful::END")
		srv.Shutdown(c)
		return nil
	})
}
