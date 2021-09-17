/*
 默认配置文件地址为 ： /home/hrx/mygo/src/CoverSer/conf/conf.yaml
*/

package conf

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type ConfInfo struct {
	Cos   CosInfo
	Db    DbInfo
	Cmq   CmqInfo
	Task  TaskInfo
	Git   GitInfo
	Log   LogConf
	Kafka KafkaInfo
}

type CosInfo struct {
	BucketUrl string `yaml:"bucket"`
	SecretId  string `yaml:"secretid"`
	SecretKey string `yaml:"secretkey"`
}

type DbInfo struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

type CmqInfo struct {
	ApiUrl    string `yaml:"apiurl"`
	SecretId  string `yaml:"secretid"`
	SecretKey string `yaml:"secretkey"`
	Queue     string `yaml:"queue"`
}

type TaskInfo struct {
	WorkSpace string `yaml:"workspace"`
}

type GitInfo struct {
	User     string `yaml:"user"`
	PassWord string `yaml:"password"`
}

type LogConf struct {
	LogDir string `yaml:"logDir"`
}

type KafkaInfo struct {
	Address      string `yaml:"address"`
	UserName     string `yaml:"username"`
	PassWord     string `yaml:"password"`
	TopicC       string `yaml:"topic-c"`
	TopicCRetry  string `yaml:"topic-c-retry"`
	TopicGo      string `yaml:"topic-go"`
	TopicGoRetry string `yaml:"topic-go-retry"`
}

var Conf ConfInfo

func Init() {
	//confPath := "/home/hrx/mygo/CoverSer/conf/conf.yaml"
	currentPath, _ := os.Getwd()
	//if _, err := os.Stat(confPath); os.IsNotExist(err) {
	//	confPath = filepath.Join(currentPath, "conf", "conf.yaml")
	//}
	confPath := filepath.Join(currentPath, "conf", "conf.yaml")
	fmt.Printf("Readaing conf file :%s", confPath)
	file, err := os.Open(confPath)
	if err != nil {
		fmt.Println("open conf error", err.Error())
		panic(fmt.Sprintf("open conf error: %s", err.Error()))
	}
	defer file.Close()

	file_b, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println("read file error")
		panic(fmt.Sprintf("read file error: %s", err.Error()))
	}
	err = yaml.Unmarshal(file_b, &Conf)
	if err != nil {
		fmt.Println("unmarshal yaml error", err)
		panic(fmt.Sprintf("unmarshal yaml error: %s", err.Error()))
	}
	if strings.HasPrefix(Conf.Task.WorkSpace, ".") || !filepath.IsAbs(Conf.Task.WorkSpace) { // 处理相对目录改成绝对路径
		Conf.Task.WorkSpace, _ = filepath.Abs(Conf.Task.WorkSpace)
	}
}
