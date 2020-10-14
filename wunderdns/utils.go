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
	"strconv"
	"strings"
	"time"
)

func generateNewSerial(oldSerial string) (newSerial string, e error) {
	oldSerial = strings.TrimFunc(oldSerial, func(r rune) bool {
		if r >= '0' && r <= '9' {
			return false
		}
		return true
	})
	serialInt, e := strconv.Atoi(oldSerial)
	currentDateSerial, e := strconv.Atoi(time.Now().Format("20060102"))
	currentDateSerial *= 100 // +2 x 00
	if serialInt >= currentDateSerial {
		serialInt++
	} else {
		serialInt = currentDateSerial
	}
	newSerial = strconv.Itoa(serialInt)
	return
}
