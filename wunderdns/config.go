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
	"fmt"
	"gopkg.in/go-ini/ini.v1"
	"strings"
	"time"
)

var globalConfig = new(Config)

var configMap = map[string]func(*ini.Section){
	"amqp":             amqpSection,
	"auth":             authSection,
	"psql":             psqlSection,
	"vault":            vaultSection,
	ini.DefaultSection: defaultSection,
}

func NewConfig(configFile string) {
	configMap["include"] = includeSection
	if e := parseConfigFile(configFile); e != nil {
		logging.Fatal("Can't load configuration file: ", e.Error())
	}
}

func parseConfigFile(configFile string) error {
	if globalConfig == nil {
		globalConfig = new(Config)
	}
	if f, e := ini.Load(configFile); e == nil {
		for k, fu := range configMap {
			_, _ = f.NewSection(k)
			if s, e := f.GetSection(k); e == nil {
				fu(s)
			}
		}
	} else {
		return e
	}
	return nil
}

func includeSection(s *ini.Section) {
	for _, sub := range s.ChildSections() {
		if sub.HasKey("file") {
			fileName := sub.Key("file").String()
			logging.Debug("Including new filename: ", fileName)
			if e := parseConfigFile(fileName); e != nil {
				logging.Error("Can't load (included) configuration file: ", e.Error())
			}
		}
	}
}

// config sample
func vaultSection(s *ini.Section) {
	if globalConfig.Vault == nil {
		globalConfig.Vault = &VaultData{
			Enabled: false,
			URL:     "",
			Token:   "",
			TTL:     10 * time.Minute,
		}
	}
	if len(s.Keys()) == 0 {
		return
	}
	if k, e := s.GetKey("enable"); e == nil {
		if globalConfig.Vault.Enabled, e = k.Bool(); e == nil && globalConfig.Vault.Enabled {
			logging.Debug("[vault] vault auth integration have been enabled")
		}
	}
	if k, e := s.GetKey("url"); e == nil {
		globalConfig.Vault.URL = k.String()
	}
	if k, e := s.GetKey("token"); e == nil {
		globalConfig.Vault.Token = k.String()
	}
	if k, e := s.GetKey("ttl"); e == nil {
		if globalConfig.Vault.TTL, e = k.Duration(); e != nil {
			globalConfig.Vault.TTL = 10 * time.Minute
		}
		logging.Debug("[vault] vault refresh time is set to ", globalConfig.Vault.TTL.Seconds(), " seconds")
	}

}

func defaultSection(s *ini.Section) {
	if s.HasKey("loglevel") {
		if k, e := s.GetKey("loglevel"); e == nil {
			if i, e := k.Int(); e == nil {
				if i >= 0 && i <= 5 {
					SetLogLevel(i)
				}
			}
		}
	}
}

func authSection(s *ini.Section) {
	authDataLock.Lock()
	defer authDataLock.Unlock()
	if globalConfig.Auth == nil {
		v := make(AuthDatabase)
		globalConfig.Auth = &v
	}
	for _, sub := range s.ChildSections() {
		a := AuthData{
			Token:       strings.TrimPrefix(sub.Name(), "auth."),
			Permissions: make([]Permission, 0),
			isVault:     false,
		}
		if sub.HasKey("secret") {
			a.Secret = sub.Key("secret").String()
			sub.DeleteKey("secret")
		} else {
			continue
		}
		if sub.HasKey("priority") {
			var e error
			if a.Priority, e = sub.Key("priority").Int(); e != nil {
				continue
			}
		}
		for _, k := range sub.Keys() {
			d := strings.Split(k.Name(), ",")
			c := strings.Split(k.String(), ",")
			p := Permission{
				Domain:    Domain{},
				Permitted: make([]Command, 0),
			}
			if len(d) == 1 {
				p.Domain.View = DomainViewAny
				p.Domain.Name = d[0]
			} else if len(d) == 2 {
				p.Domain.View = DomainView(d[0])
				p.Domain.Name = d[1]
				if x, ok := domainViews[p.Domain.View]; !(ok && x) {
					continue // bad type
				}
			}
			for _, v := range c {
				if x, ok := commands[Command(v)]; ok && x {
					p.Permitted = append(p.Permitted, Command(v))
				}
			}
			a.Permissions = append(a.Permissions, p)
		}
		(*globalConfig.Auth)[a.Token] = a
	}
}

func amqpSection(s *ini.Section) {
	if globalConfig.AMQPConfigs == nil {
		globalConfig.AMQPConfigs = make([]*AMQPConfig, 0)
	}
	for _, sub := range s.ChildSections() {
		a := AMQPConfig{}
		if sub.HasKey("url") {
			a.URL = sub.Key("url").String()
		} else {
			continue // no url - no amqp
		}
		if sub.HasKey("exchange") {
			a.Exchange = sub.Key("exchange").String()
		} else {
			continue // no exchange - no amqp
		}
		globalConfig.AMQPConfigs = append(globalConfig.AMQPConfigs, &a)
	}
}

func psqlSection(s *ini.Section) {
	if globalConfig.PSQLConfigs == nil {
		globalConfig.PSQLConfigs = make([]*PSQLConfig, 0)
	}
	for _, sub := range s.ChildSections() {
		a := PSQLConfig{SSL: false}
		if k, e := sub.GetKey("host"); e == nil {
			a.Host = k.String()
		} else {
			logging.Warning("database %s has no `host` - skipping", s.Name())
			continue
		}
		if k, e := sub.GetKey("port"); e == nil {
			a.Port, e = k.Int()
			if e != nil {
				a.Port = 5432
			}
		} else {
			a.Port = 5432
		}
		if k, e := sub.GetKey("database"); e == nil {
			a.Database = k.String()
		} else {
			logging.Warning("database %s has no `datababase` - skipping", s.Name())
			continue

		}
		if k, e := sub.GetKey("username"); e == nil {
			a.Username = k.String()
		} else {
			logging.Warning("database %s has no `username` - skipping", s.Name())
			continue
		}
		if k, e := sub.GetKey("password"); e == nil {
			a.Password = k.String()
		} else {
			a.Password = ""
		}
		if k, e := sub.GetKey("view"); e == nil {
			a.View = DomainView(k.String())
			if x, ok := domainViews[a.View]; !(x && ok) {
				continue
			}
		} else {
			if k, e := sub.GetKey("type"); e == nil {
				a.View = DomainView(k.String())
				if x, ok := domainViews[a.View]; !(x && ok) {
					continue
				}
			} else {
				logging.Warning("database %s has no `view` or `type` - skipping", s.Name())
				continue
			}
		}
		if k, e := sub.GetKey("ssl"); e == nil {
			if x, e := k.Bool(); e == nil {
				a.SSL = x
			}
		}
		globalConfig.PSQLConfigs = append(globalConfig.PSQLConfigs, &a)
	}

}

func (c *PSQLConfig) connString() string {
	opts := make([]string, 0)
	if c.Host != "" {
		opts = append(opts, fmt.Sprintf("host=%s", c.Host))
	}
	if c.Port != 5432 {
		opts = append(opts, fmt.Sprintf("port=%d", c.Port))
	}
	if c.Username != "" {
		opts = append(opts, fmt.Sprintf("user=%s", c.Username))
	}
	if c.Password != "" {
		opts = append(opts, fmt.Sprintf("password=%s", c.Password))
	}
	if c.Database != "" {
		opts = append(opts, fmt.Sprintf("dbname=%s", c.Database))
	}
	if c.SSL != true {
		opts = append(opts, "sslmode=disable")
	}

	return strings.Join(opts, " ")
}
