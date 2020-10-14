// Copyright 2018-2020 Wargaming.Net
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package httpapi

import (
	"encoding/json"
	"github.com/streadway/amqp"
	"gopkg.in/go-ini/ini.v1"
	"log"
	"math/rand"
	"github.com/wgnet/wunderdns/wunderdns"
	"strings"
	"time"
)

type producerConfig struct {
	URL      string
	Exchange string
}

func newProducer(configFile string) *producerConfig {
	ret := new(producerConfig)
	if f, e := ini.Load(configFile); e != nil {
		log.Fatal("Can't load configuration file", e.Error())
	} else {
		if s, e := f.GetSection("producer"); e != nil {
			log.Fatal("Section [producer] not found")
			return nil
		} else {
			if s.HasKey("url") {
				ret.URL = s.Key("url").String()
			} else {
				log.Fatal("url not found in section [producer]")
				return nil
			}
			if s.HasKey("exchange") {
				ret.Exchange = s.Key("exchange").String()
			} else {
				log.Fatal("exchange not found in section [producer]")
				return nil
			}
		}
	}
	return ret
}

func (conf *producerConfig) pushMessage(request *wunderdns.WunderRequest) *wunderdns.WunderReply {
	url := conf.URL
	if !strings.HasPrefix(url, "amqp://") {
		url = "amqp://" + url
	}
	conn, e := amqp.Dial(url)
	if e != nil {
		return wunderdns.ReturnError("amqp connection error: " + e.Error())
	}
	defer conn.Close()
	ch, e := conn.Channel()
	if e != nil {
		return wunderdns.ReturnError("amqp channel error: " + e.Error())
	}
	defer ch.Close()
	q, e := ch.QueueDeclare(
		"",    // name
		false, // durable
		true,  // delete when unused
		true,  // exclusive
		false, // noWait
		nil,   // arguments
	)
	if e != nil {
		return wunderdns.ReturnError("amqp queue error: " + e.Error())
	}
	defer ch.QueueDelete(q.Name, false, false, false)
	msgs, e := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if e != nil {
		return wunderdns.ReturnError("amqp consume error: " + e.Error())
	}
	data, e := json.Marshal(request)
	if e != nil {
		return wunderdns.ReturnError("json marshal error: " + e.Error())
	}
	corrId := randomString(64)
	e = ch.Publish(
		conf.Exchange,
		"",
		false,
		false,
		amqp.Publishing{
			ContentType:   "application/json",
			ReplyTo:       q.Name,
			CorrelationId: corrId,
			Body:          data,
		},
	)
	if e != nil {
		return wunderdns.ReturnError("amqp publish error: " + e.Error())
	}
	timeout := time.After(15 * time.Minute) // 15 minutes to get reply
	for {
		select {
		case m := <-msgs:
			if m.CorrelationId == corrId {
				ret := new(wunderdns.WunderReply)
				if e := json.Unmarshal(m.Body, ret); e != nil {
					return wunderdns.ReturnError("json unmarshal error: " + e.Error())
				} else {
					return ret
				}
			}
			break
		case <-timeout:
			return wunderdns.ReturnError("request timeout")
		}
	}
}

func randInt(min int, max int) int {
	return min + rand.Intn(max-min)
}

func randomString(l int) string {
	bytes := make([]byte, l)
	for i := 0; i < l; i++ {
		bytes[i] = byte(randInt(65, 90))
	}
	return string(bytes)
}
