package cmq

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"

	"CoverSer/util/conf"
)

// 腾讯云 CMQ 使用文档
// https://cloud.tencent.com/document/product/406/5837

// 向 cmq 发送 msg 内容的消息
type SendCmqRsp struct {
	Code      int
	Message   string
	RequestId string
	MsgId     string
}

func SendCmq(msg string) (error, *SendCmqRsp) {
	p := map[string]string{
		"Action":    "SendMessage",
		"queueName": conf.Conf.Cmq.Queue,
		"msgBody":   msg,
	}
	err, b := Api3Req(p)
	if err != nil {
		return err, nil
	}
	rsp := &SendCmqRsp{}
	if err := json.Unmarshal(b, rsp); err != nil {
		return err, nil
	}
	return nil, rsp
}

// 从 cmq 获取消息
type GetCmqRsp struct {
	Code          int
	Message       string
	RequestId     string
	MsgBody       string
	ReceiptHandle string
}

func GetCmq() (error, *GetCmqRsp) {
	p := map[string]string{
		"Action":    "ReceiveMessage",
		"queueName": conf.Conf.Cmq.Queue,
	}
	err, b := Api3Req(p)
	if err != nil {
		return err, nil
	}
	rsp := &GetCmqRsp{}
	if err := json.Unmarshal(b, rsp); err != nil {
		return err, nil
	}
	return nil, rsp
}

// 消息消费完成后，删除 cmq 消息记录
type DelCmqRsp struct {
	Code      int
	Message   string
	RequestId string
}

func Deletor(receiptHandle string) (error, *DelCmqRsp) {
	p := map[string]string{
		"Action":        "DeleteMessage",
		"queueName":     conf.Conf.Cmq.Queue,
		"receiptHandle": receiptHandle,
	}
	err, b := Api3Req(p)
	if err != nil {
		return err, nil
	}
	rsp := &DelCmqRsp{}
	if err := json.Unmarshal(b, rsp); err != nil {
		return err, nil
	}
	return nil, rsp
}

// 获取队列属性
func GetQueue(queue_name string) (error, string) {
	p := map[string]string{
		"Action":    "GetQueueAttributes",
		"queueName": queue_name,
	}
	err, b := Api3Req(p)
	if err != nil {
		return err, ""
	}
	return err, string(b)
}

// 修改队列取出时间
func ModifyQueue(queue_name string) (error, string) {
	p := map[string]string{
		"Action":            "SetQueueAttributes",
		"queueName":         queue_name,
		"visibilityTimeout": "300",
	}
	err, b := Api3Req(p)
	if err != nil {
		return err, ""
	}
	return err, string(b)

}

// 后面的内容为云api相关
func GetSign(p map[string]string) string {
	keys := []string{}
	for k := range p {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	plain_text := ""
	for i, v := range keys {
		if i == 0 {
			plain_text = v + "=" + p[v]
		} else {
			plain_text = plain_text + "&" + v + "=" + p[v]
		}
	}

	apiurl := conf.Conf.Cmq.ApiUrl
	plain_text = "GET" + apiurl + "/v2/index.php?" + plain_text

	key := conf.Conf.Cmq.SecretKey
	mac := hmac.New(sha1.New, []byte(key))
	mac.Write([]byte(plain_text))
	hmac_sha1 := mac.Sum(nil)
	encoded := base64.StdEncoding.EncodeToString(hmac_sha1)
	return encoded
}

func Api3Req(p map[string]string) (error, []byte) {
	apiurl := "http://" + conf.Conf.Cmq.ApiUrl + "/v2/index.php"
	data := url.Values{}
	p["Region"] = "gz"
	p["Timestamp"] = strconv.FormatInt(time.Now().Unix(), 10)
	p["Nonce"] = strconv.Itoa(rand.Intn(99999))
	p["SecretId"] = conf.Conf.Cmq.SecretId
	p["Signature"] = GetSign(p)

	for k := range p {
		data.Set(k, p[k])
	}
	req_url := apiurl + "?" + data.Encode()
	//fmt.Println(req_url)
	resp, err := http.Get(req_url)
	if err != nil {
		return err, []byte("")
	}
	defer func() {
		if resp != nil {
			resp.Body.Close()
		}
	}()
	b, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Errorf("return code is not 200,but $d, response content:%s", resp.StatusCode, string(b)), nil
	}
	//fmt.Println(string(b))
	return nil, b
}
