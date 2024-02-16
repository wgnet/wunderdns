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
package main

import (
	"flag"
	"github.com/wgnet/wunderdns/httpapi"
	"github.com/wgnet/wunderdns/wunderdns"
	"log"
	"os"
)

func main() {
	configFile := flag.String("config", "wunderdns.ini", "Config file location")
	apiConfig := flag.String("apiconfig", "wunderapi.ini", "API config file location")
	runApi := flag.Bool("api", true, "Run API server")
	runWunder := flag.Bool("wunder", true, "Run WunderDNS server")
	loglevel := flag.Int("loglevel", 3, "Loglevel: 0-5, more >> more logs")
	orm := flag.Bool("orm", true, "Enable ORM instead of SQL [deprecated]")
	flag.Parse()
	if *orm {
		wunderdns.EnableOrm()
	}
	if *runApi && *runWunder {
		go func() {
			e := httpapi.StartAPI(*apiConfig)
			if e != nil {
				log.Fatal(e.Error())
				os.Exit(1)
			}
		}()
		wunderdns.NewConfig(*configFile)
		if *loglevel >= 0 && *loglevel <= 5 && *loglevel != 3 {
			wunderdns.SetLogLevel(*loglevel) // override config
		}

		wunderdns.Run()
	} else if *runApi {
		e := httpapi.StartAPI(*apiConfig)
		if e != nil {
			log.Fatal(e.Error())
		}
	} else if *runWunder {
		wunderdns.NewConfig(*configFile)
		if *loglevel >= 0 && *loglevel <= 5 && *loglevel != 3 {
			wunderdns.SetLogLevel(*loglevel) // override config
		}
		wunderdns.Run()
	}
}
