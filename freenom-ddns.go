package main

import (
	"log"

	"freenom-ddns/internal/checkprofile"
	"freenom-ddns/internal/freenom"
	"freenom-ddns/internal/scheduler"
	"freenom-ddns/server/httpservice"
)


//zone_name="tfly.ml"   # 要做指向的根域名
//record_name="www.tfly.ml"   # 要做指向的记录

func task(f *freenom.Freenom, acs int) {
	var i int
	for i = 0; i < acs; i++ {
		f.GetIp().Login(i).RenewDomains(i) // .UpdateRecord(i,"ufly.ml","BT","45.76.105.88")
		for _, d := range f.Users[i].Domains {
			log.Println("log: ", d)
		}
	}
}

func main() {
	log.Println("Init")
	config, _ := checkprofile.ReadConf("./config.toml")
	f := freenom.GetInstance()
	f.InputAccount(config)
	task(f, len(config.Accounts))
	go scheduler.Run(func() {
		task(f, len(config.Accounts))
	}, config.System.CronTiming)

	httpservice.Run(f, config)
}
