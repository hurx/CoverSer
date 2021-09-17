package GetCover

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"

	"CoverSer/util/gorm"
	Log "CoverSer/util/log"
)

const (
	COS_DOMAIN = ""
)

type GetCoverInfo struct {
	Repo     string `form:"gitpath" json:"gitpath" binding:"required"`
	Branch   string `form:"branch" json:"branch" binding:"required"`
	CommitId string `form:"commitid" json:"commitid"`
}

func GetCover(c *gin.Context) {
	logger := Log.Logger.WithField("field", "GetCover")
	var params GetCoverInfo
	if err := c.ShouldBindJSON(&params); err != nil {
		logger.Errorf("get response body and json decode error: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    -1,
			"message": "invalid params:" + err.Error(),
		})
		return
	}
	params.Repo = strings.TrimPrefix(params.Repo, "http://")
	params.Repo = strings.TrimPrefix(params.Repo, "https://")
	params.Repo = strings.TrimSuffix(params.Repo, ".git")
	Log.Logger.Infoln("get cover:", Log.Jtos(params))
	res, err := gorm.Select(params.Repo, params.Branch, params.CommitId)
	if err != nil {
		logger.Errorf("get repo latest cover report fail: %s", err.Error())
		c.JSON(http.StatusOK, gin.H{
			"code":    -2,
			"massage": "internal error",
		})
		return
	}
	if res == nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    -3,
			"massage": "no coverage rate before",
		})
		return
	}
	res.CoverReport = COS_DOMAIN + res.CoverReport
	res.IncreaseCoverReport = COS_DOMAIN + res.IncreaseCoverReport
	res.CoverInfoFile = COS_DOMAIN + res.CoverInfoFile
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"massage": res,
	})
	return
}

func GetCoverList(c *gin.Context) {
	logger := Log.Logger.WithField("field", "GetCoverList")
	var req_list []GetCoverInfo
	if err := c.ShouldBindJSON(&req_list); err != nil {
		logger.Errorf("get response body and json decode error: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    -1,
			"message": "invalid params:" + err.Error(),
		})
		return
	}
	var resp_list []*gorm.CoverageTask
	for _, params := range req_list {
		params.Repo = strings.TrimPrefix(params.Repo, "http://")
		params.Repo = strings.TrimPrefix(params.Repo, "https://")
		params.Repo = strings.TrimSuffix(params.Repo, ".git")
		Log.Logger.Infoln("get cover:", Log.Jtos(params))
		res, err := gorm.Select(params.Repo, params.Branch, params.CommitId)
		if err != nil {
			logger.Errorf("get repo latest cover report fail: %s", err.Error())
			continue
		}
		res.CoverReport = COS_DOMAIN + res.CoverReport
		res.IncreaseCoverReport = COS_DOMAIN + res.IncreaseCoverReport
		res.CoverInfoFile = COS_DOMAIN + res.CoverInfoFile
		resp_list = append(resp_list, res)
	}
	if len(resp_list) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"code":    -2,
			"message": "no recorde before",
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": resp_list,
		})
	}
	return
}
