package DoTask

import (
	"CoverSer/util/cmd"
	"CoverSer/util/conf"
	"CoverSer/util/cos"
	"CoverSer/util/cover"
	"CoverSer/util/git"
	"CoverSer/util/gorm"
	"encoding/base64"
	"encoding/json"
	"os"
	"path"
	"sync"
	"time"

	"CoverSer/CreateTask"
	tasklog "CoverSer/util/log"
)

const (
	TIMEOUT = 30 // second
)

// 任务上下文
type TaskContext struct {
	TaskId         string // unixtime-branch
	Repository     string //
	Branch         string
	CommitId       string
	Project        string
	CreateTime     string // linux time stamp
	CoverSourceDir string
	DevLanguage    string
	WorkSpace      string
	TaskInfo       Info // 创建任务带过来的参数
	TaskDB         *gorm.CoverageTask
}

// 创建任务带来过的参数
type Info struct {
	Filter     []string // 需要过滤的目录
	Accumulate int      // 是否累积之前的数据, 0 不累计， 1 累积。 默认 1
}

func NewTask(git_path string, branch string, commit_id string, create_time string, cos_cover_file string, dev_language string) *TaskContext {
	task_id := create_time + "-" + branch
	project := path.Base(git_path)
	return &TaskContext{
		TaskId:         task_id,
		Repository:     git_path,
		Branch:         branch,
		CommitId:       commit_id,
		Project:        project,
		CreateTime:     create_time,
		CoverSourceDir: cos_cover_file,
		DevLanguage:    dev_language,
		WorkSpace:      path.Join(conf.Conf.Task.WorkSpace, git_path, branch, commit_id, create_time),
	}
}

//var log *logrus.Logger

func Chandler(task_str string) {
	//path := "/data/log/CoverSer/server.log"
	//log = Log.NewTaskLogger(path)
	task_info := &CreateTask.CreateTaskInfo{}
	if err := json.Unmarshal([]byte(task_str), task_info); err != nil {
		tasklog.TaskLogger.WithField("field", "main").Errorln("task info unmarshal err ", err.Error())
		return
	}
	task := NewTask(task_info.GitPath, task_info.Branch, task_info.CommitId, task_info.CreateTime, task_info.CoverSourceDir, task_info.DevLanguage)
	if task_info.Info != "" {
		decodeBytes, err := base64.StdEncoding.DecodeString(task_info.Info)
		if err != nil {
			tasklog.TaskLogger.WithField("field", "main").Errorln("decode base64 error ", err.Error())
		}
		info := &Info{}
		info.Accumulate = 1
		if err := json.Unmarshal(decodeBytes, info); err != nil {
			tasklog.TaskLogger.WithField("field", "main").Errorln("info unmarshal err ", err.Error())
			return
		}
		task.TaskInfo = *info
	}
	tasklog.TaskLogger.WithField("field", "main").Infof("start do task : %s\n", tasklog.Jtos(task_info))
	task.Dotask()
}

// 任务逻辑可自解释
func (t *TaskContext) Dotask() {
	tasklog.TaskLogger.WithField("field", t.TaskId).Infoln("Task start, GLHF")
	defer func() {
		cur_task := gorm.CurTask(t.Repository, t.Branch, t.CommitId, t.CreateTime)
		if (cur_task.Status != "finish") && (cur_task.Status != "timeout") {
			tasklog.TaskLogger.WithField("field", t.TaskId).Errorln("Task panic")
			panic_time := time.Now().Format("2006-01-02 15:04:05")
			gorm.UpdateStatus(cur_task, gorm.TaskStatusPanic, panic_time)
		}
	}()

	if err := t.InsertTaskIntoDB(); err != nil {
		tasklog.TaskLogger.WithField("field", t.TaskId).Errorln("insert task into db error:", err)
	}
	// 清除历史遗留目录
	history_dir := path.Join(conf.Conf.Task.WorkSpace, t.Repository, t.Branch)
	if err := os.RemoveAll(history_dir); err != nil {
		tasklog.TaskLogger.WithField("field", t.TaskId).Errorln("Delete history dir error:", err)
		return
	}
	if err := t.CreateWorkSpace(); err != nil {
		tasklog.TaskLogger.WithField("field", t.TaskId).Errorln("Create workspace error:", err)
		return
	}
	if err := t.GitPull(); err != nil {
		tasklog.TaskLogger.WithField("field", t.TaskId).Errorln("Git pull error:", err)
		return
	}
	t.WorkSpace = path.Join(t.WorkSpace, t.Project)
	if err := t.DownloadCoverSourceFile(); err != nil {
		tasklog.TaskLogger.WithField("field", t.TaskId).Errorln("Get gcda.zip and gcno.zip from cos error:", err)
		return
	}
	t.UnCoverZip(t.WorkSpace)
	if err := t.BuildCoverInfo(); err != nil {
		tasklog.TaskLogger.WithField("field", t.TaskId).Errorln("From *.gcda and *.gcno create cover.info error!")
		return
	}
	cmd.CombineInfo(t.WorkSpace, "cover.info")

	// 过滤不需要计算覆盖率的文件, 默认过滤 .h 文件 和 依赖文件
	filter := " '*.h' '/usr/*'"
	if t.TaskInfo.Filter != nil {
		for _, v := range t.TaskInfo.Filter {
			filter += (" '*" + v + "*'")
		}
	}
	tasklog.TaskLogger.WithField("field", t.TaskId).Info("Info Filter .h")
	cmd.InfoFilter(t.WorkSpace, "cover.info", "cover.info", filter)

	// 兼容 cmake , 替换 qci 的执行目录： /data/__qci/root-workspaces/__qci-pipeline-[0-9].*-[0-9]
	cmd.Replace(t.WorkSpace, "cover.info", "\\/data\\/__qci\\/root-workspaces\\/__qci-pipeline-[0-9].*-[0-9]\\/", "")

	// 生成分支覆盖率文件
	t.BranchCover("cover.info")

	// 查询当前分支之前是否有版本计算过覆盖率数据
	task_latest := gorm.Latest(t.Repository, t.Branch, t.CreateTime)
	tasklog.TaskLogger.WithField("field", t.TaskId).Info("Latest task about this repo and branch is :", task_latest)
	var cover_info_list []*cover.CoverInfo
	cover_info_file := path.Join(t.WorkSpace, "cover.info")
	// 如果之前有版本计算过覆盖率, 将之前的覆盖率数据和当前的覆盖率合并
	if task_latest != nil && task_latest.CoverRate != 0 && t.TaskInfo.Accumulate == 1 {
		tasklog.TaskLogger.WithField("field", t.TaskId).Infoln("There are commit before")
		cover_info_file_latest := path.Join(t.WorkSpace, "cover_info_latest.info")
		if err := cos.DownloadFile(task_latest.CoverInfoFile, cover_info_file_latest); err != nil {
			tasklog.TaskLogger.WithField("field", t.TaskId).Errorln("Down load latest cover info from cos error: ", err)
			return
		}
		if task_latest.CommitId == t.CommitId {
			//如果上次的 commit 和当前 commit 相同，则合并两次
			// Lcov -a a.info -a b.info -o all.info
			tasklog.TaskLogger.WithField("field", t.TaskId).Infoln("The same commit id")
			// 将上次的 cover.info 文件中的路径替换成本次相同路径
			tasklog.TaskLogger.WithField("field", t.TaskId).Infoln("Combine *.info files")
			cmd.Replace(t.WorkSpace, cover_info_file_latest, task_latest.CreateTime, t.CreateTime)
			cmd.CombineInfos(t.WorkSpace, cover_info_file_latest, cover_info_file, "all_cover.info")
			cover_info_file = path.Join(t.WorkSpace, "all_cover.info")
			if !CheckDir(cover_info_file) {
				tasklog.TaskLogger.WithField("field", t.TaskId).Error("Combine *.info files into all_cover.info error")
				return
			}
			tasklog.TaskLogger.WithField("field", t.TaskId).Infoln("Decode all_cover.info file")
			cover_info_list = cover.Decode(cover_info_file)
		} else {
			// 如果 commit id 不同，则根据 git diff 合并两次commit id
			tasklog.TaskLogger.WithField("field", t.TaskId).Infoln("Different commit id")
			// 将上次的 cover.info 文件中的路径替换成本次相同路径 - 时间戳
			tasklog.TaskLogger.WithField("field", t.TaskId).Infoln("change timestamp from last to currency")
			cmd.Replace(t.WorkSpace, cover_info_file_latest, task_latest.CreateTime, t.CreateTime)
			// 将上次的 cover.info 文件中的路径替换成本次相同路径 - commitid
			tasklog.TaskLogger.WithField("field", t.TaskId).Infoln("change commitid from last to currency")
			cmd.Replace(t.WorkSpace, cover_info_file_latest, task_latest.CommitId, t.CommitId)
			tasklog.TaskLogger.WithField("field", t.TaskId).Infoln("Combine *.info files")
			// 获取上一次的覆盖率数据
			cover_info_latest := cover.Decode(cover_info_file_latest)
			// 解析这一次的覆盖率数据
			cover_info_current := cover.Decode(cover_info_file)
			// 获取 git diff 数据
			git_diff := git.GetGitDiff(t.WorkSpace, task_latest.CommitId, t.CommitId)
			// 通过两次覆盖率数据和 git diff 数据计算最终的覆盖率数据
			cover_info_list = cover.CombineCoverInfos(t.Project, cover_info_latest, cover_info_current, git_diff)
			// 将合并后的覆盖率数据写入 cover_int.info 文件
			tasklog.TaskLogger.WithField("field", t.TaskId).Infoln("Wirte into cover_int.info")
			cover_info_file = path.Join(t.WorkSpace, "cover_int.info")
			cover.EncodeInfo(cover_info_file, cover_info_list)
		}
	} else {
		// 如果之前没有版本计算过覆盖率，将当前覆盖率info文件解析成覆盖率数据
		tasklog.TaskLogger.WithField("field", t.TaskId).Infoln("No commit before")
		cover_info_list = cover.Decode(cover_info_file)
		cover_info_file = path.Join(t.WorkSpace, "cover.info")
	}

	// 过滤
	if t.Repository == "git.code.oa.com/avcomm/trtc" {
		tasklog.TaskLogger.WithField("field", t.TaskId).Info("Info Filter .h  and  src/pb")
		cmd.InfoFilter(t.WorkSpace, cover_info_file, cover_info_file, "'*Common/src/pb/*' '*.h'")
	} else {
		tasklog.TaskLogger.WithField("field", t.TaskId).Info("Info Filter .h")
		cmd.InfoFilter(t.WorkSpace, cover_info_file, cover_info_file, " '*.h'")
	}

	// 重新解析覆盖率文件
	cover_info_list = cover.Decode(cover_info_file)

	// 全量覆盖率
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		t.CalcCoverage(cover_info_list, cover_info_file)
		wg.Done()
	}()
	// 增量覆盖率
	wg.Add(1)
	go func() {
		t.IncreaseCover(cover_info_list)
		wg.Done()
	}()
	wg.Wait()

	finish_time := time.Now().Format("2006-01-02 15:04:05")
	gorm.UpdateStatus(t.TaskDB, gorm.TaskStatusFinish, finish_time)
}
