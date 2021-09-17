// 每个步骤都应该是独立的
// 每个步骤都应该有明确校验结果
package DoTask

import (
	"errors"
	"os"
	"path"

	"CoverSer/util/cmd"
	"CoverSer/util/cos"
	"CoverSer/util/cover"
	"CoverSer/util/git"
	"CoverSer/util/gorm"
	tasklog "CoverSer/util/log"
)

// 创建任务
func (t *TaskContext) InsertTaskIntoDB() error {
	cover_db := &gorm.CoverageTask{
		TaskId:       t.TaskId,
		Repo:         t.Repository,
		Branch:       t.Branch,
		CommitId:     t.CommitId,
		CreateTime:   t.CreateTime,
		CoverDataDir: t.CoverSourceDir,
		Status:       "doing",
	}
	tasklog.TaskLogger.WithField("field", t.TaskId).Infoln("Inset task into db:", tasklog.Jtos(cover_db))
	err := gorm.Insert(cover_db)
	if err != nil {
		t.TaskDB = cover_db
		return err
	}
	t.TaskDB = cover_db
	return nil
}

// 校验文件或路径是否存在, 存在为 true， 不存在为 false
func CheckDir(dir string) bool {
	_, err := os.Stat(dir)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

// 创建任务 workspace
func (t *TaskContext) CreateWorkSpace() error {
	tasklog.TaskLogger.WithField("field", t.TaskId).Infoln("Create task workspace:", t.WorkSpace)
	if err := os.MkdirAll(t.WorkSpace, os.ModePerm); err != nil {
		return err
	}
	if !CheckDir(t.WorkSpace) {
		return errors.New("Create task workspace error!")
	}
	return nil
}

// 从 git 拉取代码并切换到指定 branch
func (t *TaskContext) GitPull() error {
	tasklog.TaskLogger.WithField("field", t.TaskId).Infoln("Git clone and checkout branch")
	cmd.GitClone(t.WorkSpace, t.Repository, t.Branch, t.CommitId)
	dir := path.Join(t.WorkSpace, t.Project)
	if !CheckDir(dir) {
		return errors.New("Git clone error:")
	}
	cmd.GitCheckout(dir, t.CommitId)
	return nil
}

// 从 cos_dir 获取 gcda.zip 和 gcno.zip, 写入本地 dir
func (t *TaskContext) DownloadCoverSourceFile() error {
	tasklog.TaskLogger.WithField("field", t.TaskId).Infoln("Get gcda and gcno file from cos.")
	cover_gcda_zip := path.Join(t.CoverSourceDir, "gcda.zip")
	cover_gcno_zip := path.Join(t.CoverSourceDir, "gcno.zip")
	cover_gcda_zip_local := path.Join(t.WorkSpace, "gcda.zip")
	cover_gcno_zip_local := path.Join(t.WorkSpace, "gcno.zip")
	if err := cos.DownloadFile(cover_gcda_zip, cover_gcda_zip_local); err != nil {
		return err
	}
	if err := cos.DownloadFile(cover_gcno_zip, cover_gcno_zip_local); err != nil {
		return err
	}
	return nil
}

// gcda.zip 和 gcno.zip 文件解压到本地
func (t *TaskContext) UnCoverZip(dir string) {
	tasklog.TaskLogger.WithField("field", t.TaskId).Infoln("Unzip gcda.zip")
	cmd.Unzip(t.WorkSpace, "gcda.zip")
	tasklog.TaskLogger.WithField("field", t.TaskId).Infoln("Unzip gcno.zip")
	cmd.Unzip(t.WorkSpace, "gcno.zip")
}

// 通过覆盖率文件生成覆盖率info文件
func (t *TaskContext) BuildCoverInfo() error {
	tasklog.TaskLogger.Infoln("From *.gcda and *.gcno create cover.info")
	// 兼容 gcno 与源文件不在同一目录
	cmd.Mvdir(t.WorkSpace, "build", "src")
	err := cmd.CreateCoverInfo(t.WorkSpace, "cover.info")
	if err != nil {
		cmd.CreateCoverInfo44(t.WorkSpace, "cover.info")
	}
	cover_info := path.Join(t.WorkSpace, "cover.info")
	if !CheckDir(cover_info) {
		return errors.New("create cover.info error!")
	}
	return nil
}

// 生成 html 的报告并上传 cos
func (t *TaskContext) CreateReport(cover_info_file string, cover_report string) error {
	tasklog.TaskLogger.WithField("field", t.TaskId).Infoln("Build html from ", cover_info_file)
	cmd.BuildHtml(t.WorkSpace, cover_info_file, cover_report)
	report_dir := path.Join(t.WorkSpace, cover_report)
	if !CheckDir(report_dir) {
		return errors.New("Create cover_report error!")
	}
	tasklog.TaskLogger.WithField("field", t.TaskId).Infoln("Upload cover html to cos")
	cos_path := path.Join(t.Repository, t.Branch, t.CommitId, t.CreateTime)
	if err := cos.UploadDir(t.WorkSpace, cover_report, cos_path); err != nil {
		new_err := errors.New(err.Error() + " local : " + cover_report + " cos: " + cos_path)
		return new_err
	}
	return nil
}

// 将 cover.info 上传到 cos
func (t *TaskContext) UploadInfoFile(cover_info_file string) error {
	tasklog.TaskLogger.WithField("field", t.TaskId).Infoln("Upload *.info to cos")
	cos_path := path.Join(t.Repository, t.Branch, t.CommitId, t.CreateTime)
	cover_info_file_cos := path.Join(cos_path, "cover.info")
	if err := cos.UploadFile(cover_info_file, cover_info_file_cos); err != nil {
		return err
	}
	return nil
}

// 计算全量覆盖率 + 生成报告 + 上传报告 + 更新数据库
func (t *TaskContext) CalcCoverage(cover_info_list []*cover.CoverInfo, cover_info_file string) {
	tasklog.TaskLogger.WithField("field", t.TaskId).Infoln("Calc coverage")
	cover_rate := cover.CoverRate(cover_info_list)

	if err := t.CreateReport(cover_info_file, "CoverReport"); err != nil {
		tasklog.TaskLogger.WithField("field", t.TaskId).Errorln("Create and upload cover_report to cos error ", err)
	}
	if err := t.UploadInfoFile(cover_info_file); err != nil {
		tasklog.TaskLogger.WithField("field", t.TaskId).Errorln("Upload cover.info error,", err)
		return
	}

	tasklog.TaskLogger.Infoln("Update task info in db")
	cos_path := path.Join(t.Repository, t.Branch, t.CommitId, t.CreateTime)
	cover_html_file_cos := path.Join(cos_path, "CoverReport/index.html")
	cover_info_file_cos := path.Join(cos_path, "cover.info")

	gorm.UpdateCover(t.TaskDB, cover_rate, cover_info_file_cos, cover_html_file_cos)
}

// 计算增量覆盖率 + 生成报告 + 上传报告 + 更新数据库
func (t *TaskContext) IncreaseCover(cover_info_list []*cover.CoverInfo) {
	tasklog.TaskLogger.Infoln("Calc increase coverage from master")
	git_diff_list := git.GetGitDiff(t.WorkSpace, "origin/master", "")
	incre_cover_rate, cover_info_incre_list := cover.GetIncreaseCover(git_diff_list, t.Project, cover_info_list, t.CommitId)
	if len(cover_info_incre_list) == 0 {
		gorm.UpdateIncreaseCover(t.TaskDB, 0, "")
		return
	}
	incre_cover_file := path.Join(t.WorkSpace, "cover_increase.info")
	cover.EncodeInfo(incre_cover_file, cover_info_incre_list)

	t.CreateReport(incre_cover_file, "increase_cover_report")

	tasklog.TaskLogger.Infoln("Update task info in db")
	cos_path := path.Join(t.Repository, t.Branch, t.CommitId, t.CreateTime)
	cover_html_file_cos := path.Join(cos_path, "increase_cover_report/index.html")

	gorm.UpdateIncreaseCover(t.TaskDB, incre_cover_rate, cover_html_file_cos)

	return
}

// 计算分支覆盖率 + 生成报告 + 上传报告
func (t *TaskContext) BranchCover(cover_info_file string) {
	tasklog.TaskLogger.WithField("field", t.TaskId).Infoln("create branch cover")
	if err := t.CreateReport(cover_info_file, "branch_cover_report"); err != nil {
		tasklog.TaskLogger.WithField("field", t.TaskId).Errorln("Create and upload branch_cover_report to cos error ", err)
		return
	}
	return
}
