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
	"log"
	"os"
)

const (
	LogFatal = iota
	LogError
	LogWarning
	LogInfo
	LogDebug
	LogTrace
)

type Logging struct {
	loglevel int
}

var logging = new(Logging)

func (l *Logging) Fatal(message ...interface{}) {
	if l.loglevel >= LogFatal {
		log.Fatal(message...)
	} else {
		os.Exit(127)
	}
}

func (l *Logging) Error(message ...interface{}) {
	if l.loglevel >= LogError {
		log.Println(append([]interface{}{"[ERROR] "}, message...)...)
	}

}

func (l *Logging) Warning(message ...interface{}) {
	if l.loglevel >= LogWarning {
		log.Println(append([]interface{}{"[WARN]  "}, message...)...)
	}
}
func (l *Logging) Info(message ...interface{}) {
	if l.loglevel >= LogInfo {
		log.Println(append([]interface{}{"[INFO]  "}, message...)...)
	}
}
func (l *Logging) Debug(message ...interface{}) {
	if l.loglevel >= LogDebug {
		log.Println(append([]interface{}{"[DEBUG] "}, message...)...)
	}
}
func (l *Logging) Trace(message ...interface{}) {
	if l.loglevel >= LogTrace {
		log.Println(append([]interface{}{"[TRACE] "}, message...)...)
	}
}

func SetLogLevel(loglevel int) {
	logging.loglevel = loglevel
	logging.Debug("Log level set to ", loglevel)
}
