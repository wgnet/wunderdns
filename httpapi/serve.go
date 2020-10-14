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
	"errors"
	"fmt"
	"github.com/asaskevich/govalidator"
	"gopkg.in/go-ini/ini.v1"
	"log"
	"net/http"
	"os"
)

var producer *producerConfig

func StartAPI(configFile string) error {
	conf := struct {
		port              int
		bind              string
		ssl               bool
		sslCertificate    string
		sslCertificateKey string
	}{}
	if f, e := ini.Load(configFile); e != nil {
		return errors.New("can't load configuration file: " + e.Error())
	} else {
		if s, e := f.GetSection("http"); e != nil {
			log.Print("[http] section not found, using default config")
			conf.port = 8080
			conf.bind = "0.0.0.0"
			conf.ssl = false
		} else {

			if s.HasKey("port") {
				if conf.port, e = s.Key("port").Int(); e != nil {
					return errors.New("http.port is not integer")
				} else if conf.port > 65535 || conf.port < 0 {
					return errors.New("http.port is not in tcp port range")
				}
			} else {
				conf.port = 8080
				log.Print("http.port is not defined, using 8080")
			}
			if s.HasKey("bind") {
				if conf.bind = s.Key("bind").String(); !govalidator.IsIP(conf.bind) {
					return errors.New("http.bind is not IP address")
				}
			} else {
				conf.bind = "0.0.0.0"
				log.Print("http.bind is not defined, using 0.0.0.0")
			}

			if s.HasKey("ssl") {
				if conf.ssl, e = s.Key("ssl").Bool(); e != nil {
					return errors.New("http.ssl is not boolean")
				}
			} else {
				conf.ssl = false
				log.Print("http.ssl is not defined, using false")
			}

			if conf.ssl {
				if s.HasKey("certificate") {
					conf.sslCertificate = s.Key("certificate").String()
					if _, e := os.Stat(conf.sslCertificate); e != nil {
						return errors.New("SSL certificate is not accessible while ssl = true")
					}
				} else {
					return errors.New("SSL certificate path is not set while ssl = true")
				}
				if s.HasKey("certificate_key") {
					conf.sslCertificateKey = s.Key("certificate_key").String()
					if _, e := os.Stat(conf.sslCertificateKey); e != nil {
						return errors.New("SSL certificate key is not accessible while ssl = true")
					}
				} else {
					return errors.New("SSL certificate key path is not set while ssl = true")
				}
			}

		}
	}
	// we're here - nice
	listen := fmt.Sprintf("%s:%d", conf.bind, conf.port)
	var e error
	producer = newProducer(configFile)
	if producer == nil {
		return errors.New("producer config not found")
	}
	// produce functions
	for x, f := range endpoints {
		http.HandleFunc(x, f)
	}
	if conf.ssl {
		e = http.ListenAndServeTLS(listen, conf.sslCertificate, conf.sslCertificateKey, nil)
	} else {
		e = http.ListenAndServe(listen, nil)
	}
	if e != nil {
		return errors.New("Serve error: " + e.Error())
	}
	return nil
}
