package cos

import (
	"CoverSer/util/conf"
	"bytes"
	"context"
	"fmt"
	"github.com/tencentyun/cos-go-sdk-v5"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
)

// 从 cos 下载文件
// cos_path  为 cos 的路径
// local_paht 为 本地保存路径
func DownloadFile(cos_path string, local_path string) error {
	u, _ := url.Parse(conf.Conf.Cos.BucketUrl)
	b := &cos.BaseURL{BucketURL: u}
	c := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  conf.Conf.Cos.SecretId,
			SecretKey: conf.Conf.Cos.SecretKey,
		},
	})

	_, err := c.Object.GetToFile(context.Background(), cos_path, local_path, nil)
	if err != nil {
		return err
	}
	return nil
}

// 上传文件到 cos
// local_path 为 本地需要上传的文件路径
// cos_path 为 cos 地址
func UploadFile(local_path string, cos_path string) error {
	u, _ := url.Parse(conf.Conf.Cos.BucketUrl)
	b := &cos.BaseURL{BucketURL: u}
	c := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  conf.Conf.Cos.SecretId,
			SecretKey: conf.Conf.Cos.SecretKey,
		},
	})

	_, err := c.Object.PutFromFile(context.Background(), cos_path, local_path, nil)
	if err != nil {
		return err
	}
	return nil
}

// upload dir
func UploadDir(workspace string, local_path string, cos_path string) error {
	file_list := []string{}
	ab_dir := path.Join(workspace, local_path)
	err := filepath.Walk(ab_dir, func(dir string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			rel_dir, err := filepath.Rel(workspace, dir)
			if err != nil {
				return err
			}
			file_list = append(file_list, rel_dir)
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, file_dir := range file_list {
		local_dir := path.Join(workspace, file_dir)
		cos_dir := path.Join(cos_path, file_dir)
		err = UploadFile(local_dir, cos_dir)
		if err != nil {
			return err
		}
	}
	return nil
}

// 遍历 cos 的 dir 目录
func ListDir(dir string) *cos.BucketGetResult {
	u, _ := url.Parse(conf.Conf.Cos.BucketUrl)
	b := &cos.BaseURL{BucketURL: u}
	client := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  conf.Conf.Cos.SecretId,
			SecretKey: conf.Conf.Cos.SecretKey,
		},
	})

	opt := &cos.BucketGetOptions{
		Prefix:    dir,
		Delimiter: "/",
	}
	rsp, _, err := client.Bucket.Get(context.Background(), opt)
	if err != nil {
		panic(err)
	}
	// obj
	// dir
	return rsp
}

// 下载 cos 整个 dir 目录至本地 local_dir 目录
func DownLoadDir(dir string, local_dir string) {
	bucket_dir := ListDir(dir)
	obj_list := bucket_dir.Contents
	for _, v := range obj_list {
		key := v.Key
		file_name := path.Base(key)
		local_path := path.Join(local_dir, file_name)
		DownloadFile(key, local_path)
	}
}

// 上传数据到cos
func UploadBytes(path string, zipDatas *bytes.Buffer) error {
	//fmt.Printf("bucketurl: %s", conf.Conf.Cos.BucketUrl)
	u, _ := url.Parse(conf.Conf.Cos.BucketUrl)
	b := &cos.BaseURL{BucketURL: u}
	c := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  conf.Conf.Cos.SecretId,
			SecretKey: conf.Conf.Cos.SecretKey,
		},
	})

	//opt := &cos.ObjectPutOptions{
	//	ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{
	//		ContentType: "application/zip",
	//	},
	//}
	//fmt.Printf("upload Path: %s \n", path)
	res, err := c.Object.Put(context.Background(), path, zipDatas, nil)
	if err != nil {
		return err
	}
	defer func() {
		res.Body.Close()
	}()
	if res.StatusCode == http.StatusOK {
		return nil
	} else {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("upload cos fail fail: %s\n", string(body))
	}
}
