package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
)

// 通过 info 文件生成覆盖率 html
// genhtml -o cover_report --legend --title "lcov"  --prefix=./ cover_int.info
func BuildHtml(dir string, info_path string, cover_report string) (error_info string) {
	prefix := "--prefix=" + dir
	var cmd *exec.Cmd
	if cover_report == "branch_cover_report" {
		cmd = exec.Command("genhtml", "-o", cover_report, "--legend", "--title", "lcov", prefix, info_path, "--ignore-errors", "source", "--branch-coverage")
	} else {
		cmd = exec.Command("genhtml", "-o", cover_report, "--legend", "--title", "lcov", prefix, info_path, "--ignore-errors", "source")
	}
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Run()
	return string(stderr.Bytes())
}

// 通过 .gcda .gconv 文件生成 .info 文件, 并用 --no-external 过滤掉 /usr/include/ 和 /usr/local/include/ 基础库的覆盖率信息
// 更新： 不再使用 --no-external 参数，因为会导致 cmake 获取不到覆盖率。 流程后面添加过 /usr/* 目录的动作。
// lcov -c -d ./ -o cover.info --no-external
// 兼容gcno 和 源文件不在同一目录 --rc geninfo_adjust_src_path="./build =>./src"  -- 该方案作废
// 添加分支覆盖率信息 --rc lcov_branch_coverage=1
// --ignore-errors source,graph 兼容源文件找不到
func CreateCoverInfo(dir string, file string) error {
	/* 方案作废
	build_dir := path.Join(dir, "build")
	src_dir := path.Join(dir, "src")
	adj := "geninfo_adjust_src_path='" + build_dir + " =>" + src_dir + "'"
	cmd := exec.Command("lcov", "-c", "-d", "./", "-o", file, "--no-external", "--rc", adj)
	*/
	//cmd := exec.Command("lcov", "-c", "-d", "./", "-o", file, "--no-external", "--rc lcov_branch_coverage=1")
	cmd_str := "lcov -c -d ./ -o " + file + "  --rc lcov_branch_coverage=1 --ignore-errors source,graph"
	cmd := exec.Command("bash", "-c", cmd_str)
	fmt.Println(cmd.String())
	var stderr bytes.Buffer
	cmd.Dir = dir
	cmd.Stderr = &stderr
	errStr := string(stderr.Bytes())
	fmt.Printf("err:\n%s\n", errStr)
	err := cmd.Run()
	return err
}

// 兼容 gcc 446 版本
func CreateCoverInfo44(dir string, file string) error {
	/* 方案作废
	build_dir := path.Join(dir, "build")
	src_dir := path.Join(dir, "src")
	adj := `geninfo_adjust_src_path="` + build_dir + "=>" + src_dir + `"`
	str := "lcov --gcov-tool /usr/bin/gcov44 -c -d " + dir + " -o " + file + " --no-external --rc " + adj
	//cmd := exec.Command("lcov", "--gcov-tool", "/usr/bin/gcov44", "-c", "-d", dir, "-o", file, "--no-external", "--rc", adj)
	cmd := exec.Command("/bin/bash", "-c", str)
	*/
	//cmd := exec.Command("lcov", "--gcov-tool", "/usr/bin/gcov44", "-c", "-d", dir, "-o", file, "--no-external", "--rc lcov_branch_coverage=1")
	cmd_str := "lcov --gcov-tool /usr/bin/gcov44 -c -d ./ -o " + file + "  --rc lcov_branch_coverage=1 --ignore-errors source,graph"
	cmd := exec.Command("bash", "-c", cmd_str)
	fmt.Println(cmd.String())
	var stderr, stdout bytes.Buffer
	cmd.Dir = dir
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	err := cmd.Run()
	//outStr, errStr := string(stdout.Bytes()), string(stderr.Bytes())
	//fmt.Printf("out:\n%s\nerr:\n%s\n", outStr, errStr)
	return err
}

// 相同 commitid 的 info 文件合并入 file
func CombineInfos(dir string, info_file_1 string, info_file_2 string, file string) {
	cmd := exec.Command("lcov", "-a", info_file_1, "-a", info_file_2, "-o", file)
	var stderr bytes.Buffer
	cmd.Dir = dir
	cmd.Stderr = &stderr
	cmd.Run()
	return
}

// 从 info 文件中过滤指定的文件路径
func InfoFilter(dir string, info_file_in string, info_file_out string, filter string) {
	cmd_str := "lcov -r " + info_file_in + " " + filter + " -o " + info_file_out + "  --rc lcov_branch_coverage=1"
	cmd := exec.Command("bash", "-c", cmd_str)
	var stderr bytes.Buffer
	cmd.Dir = dir
	fmt.Println(cmd.String())
	cmd.Stderr = &stderr
	cmd.Run()
	return
}

// 将 dir 目录下的 file 中 str1 的内容替换成 str2
func Replace(dir string, file string, str1 string, str2 string) {
	pattern := "s/" + str1 + "/" + str2 + "/g"
	cmd := exec.Command("sed", "-in-place", "-e", pattern, file)
	var stderr bytes.Buffer
	cmd.Dir = dir
	cmd.Stderr = &stderr
	cmd.Run()
	return
}

// 合并 .info 文件中的相同文件的覆盖率数据
// lcov -a cover.info -o cover.info
func CombineInfo(dir string, info_file string) {
	cmd_str := "lcov -a " + info_file + " -o " + info_file + " --rc lcov_branch_coverage=1"
	cmd := exec.Command("bash", "-c", cmd_str)
	var stderr bytes.Buffer
	cmd.Dir = dir
	cmd.Stderr = &stderr
	cmd.Run()
	return
}

// 在 dir 中执行 mv  dir1 dir2
func Mvdir(dir string, dir1 string, dir2 string) {
	dir1 = path.Join(dir, dir1)
	dir2 = path.Join(dir, dir2)
	_, dir1_err := os.Stat(dir1)
	_, dir2_err := os.Stat(dir2)
	if dir1_err == nil && dir2_err == nil {
		cmd_str := "mv " + dir1 + " " + dir2
		cmd := exec.Command("bash", "-c", cmd_str)
		cmd.Dir = dir
		cmd.Run()
		return
	}
	return
}
