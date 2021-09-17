package cmd

import (
	"bytes"
	"os/exec"
)

// 打包dir目录下的内容，压缩后的文件名问 zip_name
func Zip(dir string, zip_name string, cover_report string) {
	cmd := exec.Command("zip", "-r", zip_name, cover_report)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Run()
	log.Debugln(cmd.String())
	log.Debugln(string(stderr.Bytes()))
	return
}

// 解压 zip_name 文件，解压到 dir 目录下
func Unzip(dir string, zip_name string) {
	cmd := exec.Command("unzip", "-o", zip_name)
	cmd.Dir = dir
	var stderr bytes.Buffer

	cmd.Stderr = &stderr
	cmd.Run()
	log.Debugln(string(stderr.Bytes()))
	return
}
