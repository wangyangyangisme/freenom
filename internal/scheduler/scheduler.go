package scheduler

import (
	"github.com/jasonlvhit/gocron"
)

// Run new cron job
func Run(run func(), timing uint64) {
	cronJob := gocron.NewScheduler()
	cronJob.Every(timing).Minute().Do(run)
	<-cronJob.Start()
}
