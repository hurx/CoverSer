package CreateTask

import (
	"CoverSer/util/cos"
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"CoverSer/util/kafka"
	Log "CoverSer/util/log"
	"github.com/gin-gonic/gin"
)

type CreateTaskInfo struct {
	GitPath        string `form:"gitpath" json:"gitpath" binding:"required"`
	Branch         string `form:"branch" json:"branch" binding:"required"`
	CommitId       string `form:"commitid" json:"commitid" binding:"required"`
	CreateTime     string `form:"createtime" json:"createtime"`
	CoverSourceDir string `form:"coversourcedir" json:"coversourcedir" binding:"required"`
	DevLanguage    string `form:"dev_language" json:"dev_language"`
	CovContent     string `form:"cov_content" json:"cov_content"` // 覆盖率数据
	Info           string `form:"info" json:"info"`               // 任务带的信息，如过滤目录等
}

// 创建任务路由
// 用于接收创建任务请求，并推送至cmq
// 如果请求中不带时间戳，那么服务端添加当前时间
func CreateTask(c *gin.Context) {
	logger := Log.Logger.WithField("field", "CreateTaskWithCov")
	var create_task_info CreateTaskInfo
	if err := c.ShouldBindJSON(&create_task_info); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	create_task_info.CreateTime = strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)

	if create_task_info.DevLanguage == "" { // 默认c++
		create_task_info.DevLanguage = "c++"
	}

	// 兼容参数, git路径去掉 http 和 .git
	create_task_info.GitPath = strings.TrimPrefix(create_task_info.GitPath, "http://")
	create_task_info.GitPath = strings.TrimPrefix(create_task_info.GitPath, "https://")
	create_task_info.GitPath = strings.TrimSuffix(create_task_info.GitPath, ".git")

	logger.Infoln("create task : ", Log.Jtos(create_task_info))

	b, err := json.Marshal(create_task_info)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": -1,
			"msg":  err.Error(),
		})
		return
	}
	err = kafka.Producer("coverage-c", create_task_info.Branch, string(b))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": -1,
			"msg":  err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
	})
	return
}

// 在请求体中上报覆盖率数据
// 在服务端上传cos
func CreateTaskWithCov(ctx *gin.Context) {
	/*
		CoverSourceDir = "repo/branch/qicJobId/qciBuildNo/commit/timestamp.zip"
	*/
	logger := Log.Logger.WithField("field", "CreateTaskWithCov")
	taskInfo := CreateTaskInfo{}
	if err := ctx.ShouldBindJSON(&taskInfo); err != nil {
		logger.Errorf("decode Json error：%s\n", err.Error())
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	taskInfo.CreateTime = strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)

	if taskInfo.DevLanguage == "golang" {
		if taskInfo.CovContent == "" {
			logger.Errorf("coverage content is null")
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "coverage content is null"})
			return
		}
	}

	//fmt.Printf("coverage data: %s\n", taskInfo.CovContent)
	taskInfo.GitPath = strings.TrimPrefix(taskInfo.GitPath, "http://")
	taskInfo.GitPath = strings.TrimPrefix(taskInfo.GitPath, "https://")
	taskInfo.GitPath = strings.TrimSuffix(taskInfo.GitPath, ".git")

	zipBuffer := new(bytes.Buffer)
	zipWriter := zip.NewWriter(zipBuffer)
	defer func() {
		if zipWriter != nil {
			zipWriter.Close() // 释放buffer
		}
	}()

	fo, err := zipWriter.Create("cover.out")
	if err != nil {
		logger.Errorf("create create zip writer fail: %s", err.Error())
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal error",
		})
		return
	}
	_, err = fo.Write([]byte(taskInfo.CovContent))
	if err != nil {
		logger.Errorf("write file fail: %s", err.Error())
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal error"})
		return
	}
	// 需要close后buffer才能有数据
	if zipWriter != nil {
		zipWriter.Close()
	}
	// upload cos
	logger.Debugf("len %d", len(zipBuffer.Bytes()))
	remakeCosdir := taskInfo.CoverSourceDir
	paths := strings.Split(remakeCosdir, "/")
	paths[len(paths)-1] = fmt.Sprintf("%s.zip", taskInfo.CreateTime)
	taskInfo.CoverSourceDir = strings.Join(paths, "/")
	logger.Infof("new coverage Coverage path is:%s", taskInfo.CoverSourceDir)
	err = cos.UploadBytes(taskInfo.CoverSourceDir, zipBuffer)
	if err != nil {
		logger.Errorf("upload cos fail: %s", err.Error())
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal error"})
		return
	}
	// clean cov content before set to cmq
	taskInfo.CovContent = ""
	// add to cmq
	b, err := json.Marshal(taskInfo)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code": -1,
			"msg":  err.Error(),
		})
		return
	}
	logger.Infof("string data: %s", string(b))
	err = kafka.Producer("coverage-go", taskInfo.Branch, string(b))
	if err != nil {
		logger.Errorf("send kafka error: %s", err.Error())
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code": -1,
			"msg":  err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
	})
	return
}
