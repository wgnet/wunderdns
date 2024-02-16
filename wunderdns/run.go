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

import "time"

var checkChannel = make(chan bool, 2)

var ormOrSql = false

func Run() {
	if globalConfig.Vault.Enabled {
		// create vault sync first
		if e := globalConfig.Auth.syncVaultData(); e != nil {
			logging.Error("[vault] sync error; further sync is disabled: ", e.Error())
			globalConfig.Vault.Enabled = false
		}
		if globalConfig.Vault.Enabled {
			go func() {
				for {
					time.Sleep(globalConfig.Vault.TTL)
					if e := globalConfig.Auth.syncVaultData(); e != nil {
						logging.Warning("[vault] sync error: ", e.Error())
					}
				}
			}()
		}

	}
	// begin
	err := initORMs()
	if err != nil {
		logging.Fatal("initORMs error: ", err.Error())
	}
	i := 1
	for _, c := range globalConfig.AMQPConfigs {
		go func() {
			e := startAMQPQueue(c)
			if e != nil {
				checkChannel <- true
				logging.Fatal("startAMQPQueue error: ", e.Error())
			}
		}()
		logging.Info("Running amqp consumer #", i)
		i++
	}
	for {
		<-checkChannel
		if len(queues) == 0 {
			logging.Info("Zero queues left - exiting")
			return
		}
	}

}

func EnableOrm() {
	ormOrSql = true
}
