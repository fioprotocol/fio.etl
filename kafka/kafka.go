package kafka

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"github.com/Shopify/sarama"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"
)

type Settings struct {
	BrokerList []string `yaml:"broker_list"`
	SaslUser string `yaml:"sasl_user"`
	SaslPassword string `yaml:"sasl_password"`
}

var brokerList []string

func getConfig() *sarama.Config {
	// TODO: make this a command line option:
	f, err := os.OpenFile("auth.yml", os.O_RDONLY, 0644)
	if err != nil {
		log.Println("Please ensure auth.yml is present and contains connection information")
		log.Fatal(err)
	}
	b, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatal(err)
	}
	auth := &Settings{}
	err = yaml.Unmarshal(b, auth)
	if err != nil {
		log.Fatal(err)
	}
	config := sarama.NewConfig()
	config.Net.DialTimeout = 10 * time.Second

	config.Net.SASL.Enable = true
	config.Net.SASL.User = auth.SaslUser
	config.Net.SASL.Password = auth.SaslPassword
	brokerList = auth.BrokerList
	config.Net.SASL.Mechanism = "PLAIN"

	config.Net.TLS.Enable = true
	config.Net.TLS.Config = &tls.Config{
		InsecureSkipVerify: true,
		ClientAuth:         0,
	}
	config.Version = sarama.V2_0_0_0
	config.ClientID = `fio.etl`
	//config.Producer.Flush.Frequency = 500 * time.Millisecond
	config.Producer.Flush.MaxMessages = 20
	config.Producer.RequiredAcks = sarama.WaitForLocal
	config.Producer.Retry.Max = 10
	config.Producer.Retry.Backoff = time.Second
	return config
}

type postProcessing struct {
	BlockNum uint32 `json:"block_num"`
}

type pChan struct {
	payload []byte
	topic   string
}

func Setup(ctx context.Context, headerChan chan []byte, txChan chan []byte, rowChan chan []byte,
	miscChan chan []byte, errs chan error, wg *sync.WaitGroup) {
	wg.Add(1)

	var pause bool
	iwg := sync.WaitGroup{}
	send := func(payload []byte, channel string, producer sarama.AsyncProducer) {
		b := bytes.NewBuffer(nil)
		gz := gzip.NewWriter(b)
		_, err := gz.Write(payload)
		if err != nil {
			log.Println(err)
			return
		}
		_ = gz.Close()

		producer.Input() <- &sarama.ProducerMessage{
			Topic: channel,
			Value: sarama.ByteEncoder(b.Bytes()),
		}
		b = nil
	}

	publisher := func(c chan *pChan) {
		producer, err := sarama.NewAsyncProducer(brokerList, getConfig())
		if err != nil {
			errs <- err
			return
		}
		defer producer.Close()
		go func() {
			for err := range producer.Errors() {
				errs <- err
				return
			}
		}()
		var msg *pChan
		for {
			select {
			case <-ctx.Done():
				iwg.Done()
				return
			case msg = <-c:
				for pause {
					time.Sleep(100 * time.Millisecond)
				}
				send(msg.payload, msg.topic, producer)
				msg = nil
			}
		}
	}
	c := make([]chan *pChan, 4)
	for i := 0; i < 4; i++ {
		c[i] = make(chan *pChan)
		iwg.Add(1)
		go publisher(c[i])
	}

	var r, header, tx, account []byte
	for {
		select {
		case r = <-rowChan:
			c[0] <- &pChan{
				payload: r,
				topic: "row",
			}
			r = nil
		case header = <-headerChan:
			c[1] <- &pChan{
				payload: header,
				topic: "block",
			}
			header = nil
		case tx = <-txChan:
			c[2] <- &pChan{
				payload: tx,
				topic: "tx",
			}
			tx = nil
		case account = <-miscChan:
			c[3] <- &pChan{
				payload: account,
				topic: "misc",
			}
			account = nil
		case <-ctx.Done():
			iwg.Wait()
			wg.Done()
			return
		}
	}
}
