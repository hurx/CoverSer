package gorm

import (
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"time"

	"CoverSer/util/conf"
)

const (
	TaskStatusDoing   = "doing"
	TaskStatusFinish  = "finish"
	TaskStatusTimeOut = "timeout"
	TaskStatusPanic   = "panic"
	MAX_POOL_SIZE     = 50
)

var DbPoll *gorm.DB

// 数据库中覆盖率的结构体
type CoverageTask struct {
	TaskId              string  `column:"task_id" type:"varchar(100)"`
	Repo                string  `column:"repo" type:"varchar(200)"`
	Branch              string  `column:"branch" type:"varchar(100)"`
	CommitId            string  `column:"commit_id" type:"varchar(50)"`
	CreateTime          string  `column:"create_time" type:"varchar(20)"`
	FinishTime          string  `column:"finish_time" type:"string"`
	CoverRate           float64 `column:"cover_rate" type:"varchar(10)"`
	IncreaseCoverRate   float64 `column:"increase_cover_rate" type:"varchar(10)"`
	CoverReport         string  `column:"cover_report" type:"varchar(300)"`
	IncreaseCoverReport string  `column:"increase_cover_report" type:"varchar(300)"`
	CoverInfoFile       string  `column:"cover_info_file" type:"varchar(300)"`
	CoverDataDir        string  `column:"cover_data_dir" type:"varchar(300)"`
	Status              string  `column:"status"  type:"varchar(100)"` // doing, finish , timeout , panic
}

func (CoverageTask) TableName() string {
	return "coverage_tasks"
}

func Init() {
	var err error
	host := conf.Conf.Db.Host
	port := conf.Conf.Db.Port
	username := conf.Conf.Db.User
	password := conf.Conf.Db.Password
	dbname := conf.Conf.Db.Database
	connect_info := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local", username, password, host, port, dbname)
	DbPoll, err = gorm.Open("mysql", connect_info)
	if err != nil {
		panic(fmt.Sprintf("create mysql Connection error: %s", err.Error()))
	}
	err = DbPoll.DB().Ping()
	if err != nil {
		panic(fmt.Errorf("ping mysql fail, msg: %s \n", err.Error()))
	}
	if DbPoll == nil {
		panic("create mysql connection fail")
	}

	DbPoll.DB().SetMaxIdleConns(20)            // 保持多少个闲置连接
	DbPoll.DB().SetMaxOpenConns(MAX_POOL_SIZE) // 最多少个连接
	DbPoll.DB().SetConnMaxLifetime(5 * time.Minute)
}

func NewDB() *gorm.DB {
	host := conf.Conf.Db.Host
	port := conf.Conf.Db.Port
	username := conf.Conf.Db.User
	password := conf.Conf.Db.Password
	dbname := conf.Conf.Db.Database
	connect_info := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local", username, password, host, port, dbname)
	db, err := gorm.Open("mysql", connect_info)
	if err != nil {
		fmt.Println("connect mysql error", err.Error())
		return nil
	}
	// if defer the db closed after init
	// defer db.Close()

	//db.CreateTable(&CoverageTask{})
	return db
}

func Insert(c *CoverageTask) error {
	//db := NewDB()
	//defer db.Close()
	return DbPoll.Create(c).Error

}

// 更新 全量覆盖率、覆盖率 info 文件 cos 地址 、覆盖率 report 文件 cos 地址
func UpdateCover(cover_db *CoverageTask, cover_rate float64, cover_info_file string, cover_report string) error {
	//db := NewDB()
	//defer db.Close()
	return DbPoll.Model(cover_db).Where("repo = ? AND branch = ? AND commit_id = ? AND create_time = ?", cover_db.Repo, cover_db.Branch, cover_db.CommitId, cover_db.CreateTime).Updates(CoverageTask{CoverRate: cover_rate, CoverInfoFile: cover_info_file, CoverReport: cover_report}).Error

}

func UpdateCoverWithTaskInfo(repo_, branch_, commit_, create_time_ string, cover_rate float64, cover_info_file string, cover_report string) error {
	//db := NewDB()
	//defer db.Close()
	return DbPoll.Model(&CoverageTask{}).Where(&CoverageTask{Repo: repo_, Branch: branch_, CommitId: commit_, CreateTime: create_time_}).Updates(CoverageTask{CoverRate: cover_rate, CoverInfoFile: cover_info_file, CoverReport: cover_report}).Error

}

// 更新 增量覆盖率 和 增量覆盖率cos地址
func UpdateIncreaseCover(cover_db *CoverageTask, increase_cover_rate float64, increase_cover_report string) error {
	//db := NewDB()
	//defer db.Close()
	return DbPoll.Model(cover_db).Where("repo = ? AND branch = ? AND commit_id = ? AND create_time = ?", cover_db.Repo, cover_db.Branch, cover_db.CommitId, cover_db.CreateTime).Updates(CoverageTask{IncreaseCoverRate: increase_cover_rate, IncreaseCoverReport: increase_cover_report}).Error

}

func UpdateIncreaseCoverWithTaskInfo(repo_, branch_, commit_, create_time_ string, increase_cover_rate float64, increase_cover_report string) error {
	//db := NewDB()
	//defer db.Close()
	return DbPoll.Model(&CoverageTask{}).Where(&CoverageTask{Repo: repo_, Branch: branch_, CommitId: commit_, CreateTime: create_time_}).Updates(CoverageTask{IncreaseCoverRate: increase_cover_rate, IncreaseCoverReport: increase_cover_report}).Error

}

// 查询当前 commit 是否有执行记录
func Select(repo string, branch string, commit_id string) (c *CoverageTask, err error) {
	//db := NewDB()
	//defer db.Close()
	c = &CoverageTask{}
	//err = db.Where("repo = ? AND branch = ? AND commit_id = ? AND status = ?", repo, branch, commit_id, "finish").Order("create_time desc").First(&c).Error
	err = DbPoll.Where(&CoverageTask{Repo: repo, Branch: branch, CommitId: commit_id, Status: TaskStatusFinish}).Order("create_time desc").First(&c).Error
	if err != nil && err.Error() == "record not found" {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

// 查询当前分支上一条任务记录，排除当前任务记录， 且 coverage 不能为 0
func Latest(repo string, branch string, create_time string) (c *CoverageTask) {
	//db := NewDB()
	//defer db.Close()
	c = &CoverageTask{}
	notFound := DbPoll.Where("repo = ? AND branch = ?  AND create_time < ? AND cover_rate <> 0", repo, branch, create_time).Order("create_time desc").First(c).RecordNotFound()
	if notFound {
		return nil
	}
	return c
}

// 查询当前任务状态
func CurTask(repo string, branch string, commit_id string, create_time string) (c *CoverageTask) {
	//db := NewDB()
	//defer db.Close()
	c = &CoverageTask{}
	DbPoll.Where("repo = ? AND branch = ? AND commit_id= ? AND create_time = ? ", repo, branch, commit_id, create_time).First(c)
	return c
}

// 更新任务状态
func UpdateStatus(task *CoverageTask, status string, finish_time string) (err error) {
	//db := NewDB()
	//defer db.Close()
	err = DbPoll.Model(task).Where("repo = ? AND branch = ? AND commit_id = ? AND create_time = ?", task.Repo, task.Branch, task.CommitId, task.CreateTime).Update(CoverageTask{Status: status, FinishTime: finish_time}).Error
	return err
}

func UpdateTaskStatusWithTaskInfo(repo_, branch_, commit_, create_time_, status_, finish_time string) error {
	//db := NewDB()
	//defer db.Close()
	return DbPoll.Model(&CoverageTask{}).Where(&CoverageTask{Repo: repo_, Branch: branch_, CommitId: commit_, CreateTime: create_time_}).Update(CoverageTask{Status: status_, FinishTime: finish_time}).Error
}
