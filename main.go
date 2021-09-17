package main

import (
	"CoverSer/util/conf"
	orm "CoverSer/util/gorm"
	"CoverSer/util/log"
	"github.com/gin-gonic/gin"

	"CoverSer/CreateTask"
	"CoverSer/GetCover"
)

func Init() {
	conf.Init()
	log.Init()
	orm.Init()
}

func main() {
	Init()
	log.Logger.WithField("field", "main").Info("CoverSer start")
	router := gin.Default()
	router.Use(log.LoggerHandler())
	router.POST("/createtask", CreateTask.CreateTask)
	router.POST("/createTaskCov", CreateTask.CreateTaskWithCov) // with cov content
	router.POST("/getcoverage", GetCover.GetCover)
	router.POST("/getcoveragelist", GetCover.GetCoverList)
	router.Run(":3001")
}
