package cmd

import (
	Log "CoverSer/util/log"
	"github.com/sirupsen/logrus"
)

var log *logrus.Logger

func init() {
	log_path := "/data/log/CoverSer/server.log"
	if log == nil {
		log = Log.NewTaskLogger(log_path)
	}
}
