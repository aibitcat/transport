package kafka

import (
	"github.com/Shopify/sarama"
	"github.com/luopengift/golibs/logger"
	"github.com/luopengift/transport/pipeline"
	"strconv"
	"strings"
	"sync"
)

/*
一些使用说明:
sarame.OffsetNewest int64 = -1
sarame.OffsetOldest int64 = -2
*/
type Consumer struct {
	Addrs   []string
	Topic   string
	Offset  int64
	Message chan []byte //从这个管道中读取数据
}

func NewConsumer(addrs []string, topic string, offset int64) *Consumer {
	return &Consumer{
		Addrs:   addrs,
		Topic:   topic,
		Offset:  offset,
		Message: make(chan []byte),
	}
}

func NewKafkaInput() *Consumer {
	c := new(Consumer)
	c.Message = make(chan []byte)
	return c
}

func (c *Consumer) Init(config map[string]string) error {
	c.Addrs = strings.Split(config["addrs"], ",")
	c.Topic = config["topic"]
	c.Offset, _ = strconv.ParseInt(config["offset"], 10, 64)
	return nil
}

func (self *Consumer) Read(p []byte) (cnt int, err error) {
	msg := <-self.Message
	n := copy(p, msg)
	return n, nil
}

func (self *Consumer) Start() error {
	self.ReadFromTopic()
	return nil
}

func (self *Consumer) ReadFromTopic() {
	var wg sync.WaitGroup
	consumer, err := sarama.NewConsumer(self.Addrs, sarama.NewConfig())
	if err != nil {
		logger.Error("<new consumer error> %v", err)
	}
	partitionList, err := consumer.Partitions(self.Topic)
	if err != nil {
		logger.Error("<consumer partitions> %v", err)
	}
	for partition := range partitionList {
		pc, err := consumer.ConsumePartition(self.Topic, int32(partition), self.Offset)
		if err != nil {
			logger.Error("<consume error> %v", err)
		}
		defer pc.AsyncClose()

		wg.Add(1)
		go func(pc sarama.PartitionConsumer) {
			defer wg.Done()
			for msg := range pc.Messages() {
				self.Message <- msg.Value
			}
		}(pc)

	}
	wg.Wait()
	consumer.Close()
}

func (self *Consumer) Close() error {
	return self.Close()
}

func init() {
	pipeline.RegistInputer("kafka", NewKafkaInput())
}