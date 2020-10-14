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
package wunderdns

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/streadway/amqp"
	"log"
	"strings"
)

type amqpqueue struct {
	conn        *amqp.Connection
	channel     *amqp.Channel
	exitChannel chan bool
}

var queues = make(map[string]*amqpqueue)

func startAMQPQueue(config *AMQPConfig) error {
	url := config.URL
	if !strings.HasPrefix(url, "amqp://") {
		url = "amqp://" + url
	}
	conn, e := amqp.Dial(url)
	if e != nil {
		return errors.New("amqp connection error: " + e.Error())
	}
	defer conn.Close()
	ch, e := conn.Channel()
	if e != nil {
		return errors.New("amqp channel error: " + e.Error())
	}
	defer ch.Close()
	e = ch.ExchangeDeclare(
		config.Exchange,
		amqp.ExchangeFanout,
		true,
		false,
		false,
		false,
		nil,
	)
	if e != nil {
		return errors.New("amqp exchangeDeclare error: " + e.Error())
	}
	q, e := ch.QueueDeclare(
		"",    // name
		false, // durable
		false, // delete when unused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if e != nil {
		return errors.New("amqp queueDeclare error: " + e.Error())
	}
	e = ch.QueueBind(
		q.Name,          // queue name
		"",              // routing key
		config.Exchange, // exchange
		false,
		nil)
	if e != nil {
		return errors.New("amqp queueBind error: " + e.Error())
	}
	msgs, e := ch.Consume(q.Name, "", true, false, false, false, nil)
	if e != nil {
		return errors.New("amqp consume error: " + e.Error())
	}
	v := make(chan bool, 2)
	queues[q.Name] = &amqpqueue{
		conn:        conn,
		channel:     ch,
		exitChannel: v,
	}
	defer func() {
		delete(queues, q.Name)
		checkChannel <- true
	}()
	for {
		select {
		case msg := <-msgs:
			go processMessage(&msg, q.Name)
		case <-v:
			log.Printf("Got quit signal!")
			return nil
		}
	}
}

func replyError(message *amqp.Delivery, key string, data ...interface{}) {
	log.Print(append([]interface{}{"ERROR:"}, data...)...)
	if message.ReplyTo != "" && message.CorrelationId != "" {
		if j, e := json.Marshal(ReturnError(data...)); e == nil {
			re := amqp.Publishing{
				ContentType:   "application/json",
				Body:          j,
				CorrelationId: message.CorrelationId,
			}
			if v, ok := queues[key]; ok {
				e := v.channel.Publish("", message.ReplyTo, false, false, re)
				if e != nil {
					logging.Warning(fmt.Sprintf("Error sending reply to %s/%s: %s", message.ReplyTo, message.CorrelationId, e.Error()))
				} else {
					logging.Trace(fmt.Sprintf("Sent reply to %s/%s: %d", message.ReplyTo, message.CorrelationId, len(j)))
				}
			}
		}
	}
}

func replySuccessData(message *amqp.Delivery, key string, data interface{}) {
	if message.ReplyTo != "" && message.CorrelationId != "" {
		if j, e := json.Marshal(ReturnSuccess(data)); e == nil {
			re := amqp.Publishing{
				ContentType:   "application/json",
				Body:          j,
				CorrelationId: message.CorrelationId,
			}
			if v, ok := queues[key]; ok {
				e := v.channel.Publish("", message.ReplyTo, false, false, re)
				if e != nil {
					logging.Warning(fmt.Sprintf("Error sending reply to %s/%s: %s", message.ReplyTo, message.CorrelationId, e.Error()))
				} else {
					logging.Trace(fmt.Sprintf("Sent reply to %s/%s: %d", message.ReplyTo, message.CorrelationId, len(j)))
				}
			}
		}
	}
}

func ReturnError(err ...interface{}) *WunderReply {
	return &WunderReply{
		Status: "ERROR",
		Data: map[string]string{
			"error": fmt.Sprint(err...),
		},
	}
}

func ReturnSuccess(data interface{}) *WunderReply {
	return &WunderReply{
		Status: "SUCCESS",
		Data:   data,
	}
}

func processMessage(message *amqp.Delivery, key string) {
	req := new(WunderRequest)
	if e := json.Unmarshal(message.Body, req); e != nil {
		replyError(message, key, "json: ", e.Error())
		return
	}
	if e := securityProcessRequest(req); e != nil {
		replyError(message, key, "security: ", e.Error())
		return
	}
	if e := checkRFCRequest(req); e != nil {
		replyError(message, key, "rfc1034: ", e.Error())
		return
	}
	logging.Trace(fmt.Sprintf("Got request (%s) from %s/%s", req.Cmd, message.ReplyTo, message.CorrelationId))
	switch req.Cmd {
	case CommandListRecords, CommandListDomains, CommandListOwn:
		d, e := applyCommandMerge(req)
		if e != nil {
			replyError(message, key, "sql: ", e.Error())
		} else {
			replySuccessData(message, key, d)
		}
		break
	default:
		if n, e := applyCommand(req); e != nil {
			replyError(message, key, "sql: ", e.Error())
			return
		} else {
			replySuccessData(message, key, map[string]interface{}{"rows": n})
		}
	}
}
