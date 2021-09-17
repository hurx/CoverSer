package main

import (
	"CoverSer/DoTask"
	"CoverSer/util/conf"
	orm "CoverSer/util/gorm"
	"CoverSer/util/kafka"
	tasklog "CoverSer/util/log"
)

const (
	TOPIC   string = "coverage-c"
	GROUP   string = "group-c"
	LOGFILE string = "c_handler.log"
)

func main() {
	conf.Init()
	tasklog.NewTaskLogger(LOGFILE)
	orm.Init()
	tasklog.TaskLogger.WithField("field", "main").Info("c handler start")
	kafka.Comsumer(conf.Conf.Kafka, TOPIC, GROUP, DoTask.Chandler)
}
