package main

import (
	"log"

	"freenom-ddns/internal/checkprofile"
	"freenom-ddns/internal/freenom"
	"freenom-ddns/internal/scheduler"
	"freenom-ddns/server/httpservice"
)

func task(f *freenom.Freenom) {

	for _,v := range f.Users {
		v.DomainList().RenewDomains()

	}
}

func ddns(f *freenom.Freenom) {

	for _,v := range f.Users {
		v.DomainList().UpdateRecord(v.ZoneName,v.RecordName)
	}
}

func main() {
	log.Println("Init")
	config, _ := checkprofile.ReadConf("./config.toml")
	f := freenom.GetInstance()
	f.InputAccount(config)
	task(f)
	ddns(f)
	go scheduler.Run(func() {
		task(f)
	}, config.System.ReNewTiming)

	go scheduler.Run(func() {
		ddns(f)
	}, config.System.DdnsTiming)

	httpservice.Run(f, config)
}
