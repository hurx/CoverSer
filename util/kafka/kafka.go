package kafka

import (
	"context"
	"fmt"
	"log"
	"time"

	"CoverSer/util/conf"
	kafka "github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"
)

func Producer(topic string, key string, value string) error {
	mechanism := plain.Mechanism{
		Username: conf.Conf.Kafka.UserName,
		Password: conf.Conf.Kafka.PassWord,
	}
	dialer := &kafka.Dialer{
		Timeout:       10 * time.Second,
		DualStack:     true,
		SASLMechanism: mechanism,
	}
	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  []string{conf.Conf.Kafka.Address},
		Topic:    topic,
		Balancer: &kafka.Hash{},
		Dialer:   dialer,
	})
	msg := kafka.Message{
		Key:   []byte(key),
		Value: []byte(value),
	}
	err := w.WriteMessages(context.Background(), msg)
	if err != nil {
		return err
	}
	defer func() {
		if w != nil {
			w.Close()
		}
	}()
	return nil
}

// 消费指定 topic， 需要指定 groupid
// 获取到的信息并发执行 dotask，并发使用 tracks chan控制
func Comsumer(kafka_info conf.KafkaInfo, topic string, group_id string, dotask func(task_info string)) {
	tracks := make(chan string, 4)
	mechanism := plain.Mechanism{
		Username: kafka_info.UserName,
		Password: kafka_info.PassWord,
	}
	dialer := &kafka.Dialer{
		Timeout:       10 * time.Second,
		DualStack:     true,
		SASLMechanism: mechanism,
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{kafka_info.Address},
		GroupID: group_id,
		Topic:   topic,
		Dialer:  dialer,
	})
	ctx := context.Background()
	go func() {
		for task_info := range tracks {
			dotask(task_info)
		}
	}()
	for {
		fmt.Println("read from kafka")
		m, err := r.ReadMessage(ctx)
		if err != nil {
			log.Println("read kafka error: ", err)
		}
		//log.Printf("message at topic/partition/offset %v/%v/%v: %s = %s\n", m.Topic, m.Partition, m.Offset, string(m.Key), string(m.Value))
		fmt.Println(len(tracks))
		tracks <- string(m.Value)
	}

	if err := r.Close(); err != nil {
		panic("failed to close  kafka reader:" + err.Error())
	}
}
