// Copyright 2018-2023 Wargaming.Net
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package wunderdns

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

func (authDatabase *AuthDatabase) syncVaultData() error {
	if !globalConfig.Vault.Enabled {
		return errors.New("vault integration is disabled")
	}
	authDataLock.Lock()
	defer authDataLock.Unlock()
	cli := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			TLSHandshakeTimeout:   10 * time.Second,
			DisableKeepAlives:     true,
			DisableCompression:    true,
			IdleConnTimeout:       10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			ExpectContinueTimeout: 10 * time.Second,
		},
		Timeout: 10 * time.Second,
	}
	var req *http.Request
	var resp *http.Response
	var e error
	tokens := make([]string, 0)
	// stage 0: RENEW TOKEN ( self )
	renewUrlParts := strings.Split(globalConfig.Vault.URL, "/")
	var renewUrl string
	for i, k := range renewUrlParts {
		if k == "v1" {
			renewUrl = strings.Join(renewUrlParts[0:i], "/") // [x:y) - remember it :)
			renewUrl += "/v1/auth/token/renew-self"
			break
		}
	}
	if renewUrl != "" {
		req, e = http.NewRequest("POST", renewUrl, nil)
		if e == nil {
			req.Header.Set("X-Vault-Token", globalConfig.Vault.Token)
			_, _ = cli.Do(req)
		}
	}
	// Stage one: make LIST request to URL
	// create LIST request on URL
	listUrl := globalConfig.Vault.URL
	if listUrl[len(listUrl)-1] != '/' {
		listUrl += "/"
	}
	req, e = http.NewRequest("LIST", listUrl, nil)
	if e != nil {
		return e
	}
	req.Header.Set("X-Vault-Token", globalConfig.Vault.Token)
	if resp, e = cli.Do(req); e != nil {
		return e
	}
	listData := make(map[string]interface{})
	if e = json.NewDecoder(resp.Body).Decode(&listData); e != nil {
		return e
	}
	if data, ok := listData["data"]; ok {
		if mdata, ok := data.(map[string]interface{}); ok {
			if keys, ok := mdata["keys"]; ok {
				if akeys, ok := keys.([]interface{}); ok {
					for _, key := range akeys {
						if skey, ok := key.(string); ok {
							tokens = append(tokens, skey)
						}
					}
				} else {
					return errors.New("[vault.syncVaultData] LIST: keys is not an array")
				}
			} else {
				return errors.New("[vault.syncVaultData] LIST: keys not found in data")
			}
		} else {
			return errors.New("[vault.syncVaultData] LIST: data field is not an object")
		}
	} else {
		return errors.New("[vault.syncVaultData] LIST: data field not found in response")
	}
	logging.Debug("[vault.requestData] got ", len(tokens), " tokens to proceed")

	// Stage two: for every TOKEN create a GET request to URL+TOKEN
	tempTokens := make(map[string]AuthData)
	for _, token := range tokens {
		newUrl := listUrl + token
		req, e = http.NewRequest("GET", newUrl, nil)
		if e != nil {
			logging.Warning("[vault.syncVaultData] ignoring ", token, ": ", e.Error())
			continue
		}
		req.Header.Set("X-Vault-Token", globalConfig.Vault.Token)
		if resp, e = cli.Do(req); e != nil {
			logging.Warning("[vault.syncVaultData] request error for token ", token, ": ", e.Error())
			continue
		}
		authData := make(map[string]interface{})
		if e = json.NewDecoder(resp.Body).Decode(&authData); e != nil {
			logging.Warning("[vault.syncVaultData] json error error for token ", token, ": ", e.Error())
			continue
		}
		if data, ok := authData["data"]; ok {
			if mdata, ok := data.(map[string]interface{}); ok {
				newAuth := AuthData{
					Token:       token,
					Permissions: make([]Permission, 0),
					isVault:     true,
				}
				for k, v := range mdata {
					if _, ok := v.(string); !ok {
						continue
					}
					if k == "secret" {
						newAuth.Secret = v.(string)
						continue
					}
					d := strings.Split(k, ",")
					c := strings.Split(v.(string), ",")
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
					newAuth.Permissions = append(newAuth.Permissions, p)
				}
				tempTokens[token] = newAuth
				logging.Debug("[vault.syncVaultData] got new auth (", token, ") with ", len(newAuth.Permissions), " permissions ")
			} else {
				logging.Warning("[vault.syncVaultData] data not found for token ", token, ": ")
				continue
			}
		} else {
			logging.Warning("[vault.syncVaultData] data is not an object for token ", token, ": ")
			continue
		}
	}
	// replace tempTokens
	for k, v := range *authDatabase {
		if v.isVault {
			if v, ok := tempTokens[k]; ok {
				(*globalConfig.Auth)[k] = v // update data
			} else {
				// deleted token
				defer delete(*authDatabase, k)
			}
		}
	}
	for k, v := range tempTokens {
		if _, ok := (*authDatabase)[k]; !ok {
			(*authDatabase)[k] = v
		}
	}
	return nil
}
