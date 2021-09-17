package cmd

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"

	"CoverSer/util/conf"
)

// git clone  代码至本地
// git path  :  git.code.oa.com/renxinhu/CoverSer.git ，不带http
// dis_dir : 存入目标目录
// 返回 :  项目文件路径
func GitClone(dis_dir string, repository string, branch string, commit_id string) error {
	user := conf.Conf.Git.User
	password := conf.Conf.Git.PassWord
	password = url.QueryEscape(password)
	git_url := "http://" + user + ":" + password + "@" + repository + ".git"
	cmd := exec.Command("git", "clone", "-b", branch, git_url)
	//fmt.Println(cmd.String())
	cmd.Dir = dis_dir
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return err
	//log.Debugln(string(stderr.Bytes()))
}

// 切换到指定 commit_id
func GitCheckout(dir string, commit_id string) {
	//cmd := exec.Command("git", "checkout", "-b", commit_id)
	cmd := exec.Command("git", "reset", "--hard", commit_id)
	cmd.Dir = dir
	stdout := bytes.Buffer{}
	stderr := bytes.Buffer{}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Run()
	//log.Debugln(string(stderr.Bytes()))
	return
}

// 通过两个commit_id生成 git diff 信息的文件
func GitDiff(dir string, commit1 string, commit2 string) (git_diff_file string) {
	var cmd *exec.Cmd
	if commit2 == "" {
		cmd = exec.Command("git", "diff", commit1)
	} else {
		cmd = exec.Command("git", "diff", commit1, commit2)
	}
	cmd.Dir = dir
	if len(commit1) > 8 {
		commit1 = commit1[len(commit1)-8:]
	}
	if len(commit2) > 8 {
		commit2 = commit2[len(commit2)-8:]
	}
	commit1 = strings.Replace(commit1, "/", "_", -1)
	file_name := "git_diff_" + commit1 + "-" + commit2 + ".log"
	file_path := path.Join(dir, file_name)
	outfile, err := os.Create(file_path) //, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0755)
	if err != nil {
		panic(err.Error())
	}
	defer outfile.Close()

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = outfile
	err = cmd.Run()
	//log.Infoln(string(stderr.Bytes()))
	git_file_path := path.Join(dir, file_name)
	return git_file_path
}

// 获取master的上一次commitid
func GitPreCommit(dir string) (preCommitId string) {
	log.Infof("dir: %s, get pre commit id", dir)
	cmd := exec.Command("git", "rev-list", "origin/master", "-n", "1", "--skip=1")
	cmd.Dir = dir
	file_name := "git_rev_list_1.log"
	file_path := path.Join(dir, file_name)
	outfile, err := os.OpenFile(file_path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0755)
	if err != nil {
		panic(err)
	}
	defer outfile.Close()

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = outfile
	err = cmd.Run()
	//log.Infoln(string(stderr.Bytes()))
	git_file_path := path.Join(dir, file_name)
	return git_file_path
}

// 将master的变更同步到本地分支 会存在失败，会产生冲突
func GitRebaseMaster(projectDir_ string) (err error) {
	log.Infof("do 'git rebase origin/master' on %s", projectDir_)
	cmd := exec.Command("git", "rebase", "origin/master")
	cmd.Dir = projectDir_
	file_name := "git_rebase_origin_master.log"
	file_path := path.Join(projectDir_, file_name)
	outfile, err := os.OpenFile(file_path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("open git resabse log fail,err:%s", err.Error())
	}
	defer outfile.Close()

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = outfile
	err = cmd.Run()
	//log.Infoln(string(stderr.Bytes()))
	if len(stderr.Bytes()) != 0 {
		return fmt.Errorf("do git rebase origin/master fail")
	}
	return nil
}
