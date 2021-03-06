package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/Shopify/sarama"
)

var (
	SARAMA_KAFKA_PROTO_VER = sarama.V0_10_2_0
)

func main() {

	var (
		messages int
		pause    time.Duration
		broker   string
		topic    string
	)

	flag.IntVar(&messages, "messages", 1, "specify the number of messages")
	flag.StringVar(&broker, "broker", "kafka:9092", "broker address and port")
	flag.StringVar(&topic, "topic", "faas-request", "topic to produce messages on")
	flag.DurationVar(&pause, "pause", time.Millisecond*100, "pause in Golang duration format")

	flag.Parse()

	brokerList := []string{broker}

	config := sarama.NewConfig()
	config.Version = SARAMA_KAFKA_PROTO_VER
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 2
	config.Producer.Return.Successes = true

	fmt.Println("Creating producer")
	producer, err := sarama.NewSyncProducer(brokerList, config)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := producer.Close(); err != nil {
			panic(err)
		}
	}()

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder("Test the function."),
	}

	log.Printf("Sending %d messages.\n", messages)
	for i := 0; i < messages; i++ {
		partition, offset, err := producer.SendMessage(msg)
		if err != nil {
			panic(err)
		}

		log.Printf("Msg: topic(%s)/partition(%d)/offset(%d)\n", topic, partition, offset)

		time.Sleep(pause)
	}
}
