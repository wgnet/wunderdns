// Copyright 2018-2020 Wargaming.Net
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
package httpapi

import (
	"crypto"
	"encoding/json"
	"fmt"
	"github.com/wgnet/wunderdns/wunderdns"
	"net/http"
	"strings"
	"time"
)

var endpoints = map[string]func(http.ResponseWriter, *http.Request){
	"/ping":    apiPingFunc,
	"/domain":  apiDomainFunc,
	"/record":  apiRecordFunc,
	"/migrate": apiMigrateFunc,
}

func writeJson(w http.ResponseWriter, r *http.Request, data interface{}) {
	if data == nil {
		writeJsonE(w, r, 503, "Internal Server Error")
	} else {
		writeJsonE(w, r, 200, data)
	}
}

func writeJsonE(w http.ResponseWriter, r *http.Request, code int, data interface{}) {
	enc := json.NewEncoder(w)
	pretty := false
	if _, ok := r.URL.Query()["pretty"]; ok {
		pretty = true
	}
	w.Header().Add("Content-View", "application/json")
	w.WriteHeader(code)
	if pretty {
		enc.SetIndent(" ", " ")
	}
	switch data.(type) {
	case string:
		data = wunderdns.ReturnError(data)
	case []string:
		data = wunderdns.ReturnError(data.([]interface{})...)
	}
	enc.Encode(data)
}

func apiPingFunc(w http.ResponseWriter, r *http.Request) {
	writeJson(w, r, wunderdns.ReturnSuccess("PONG"))
}

func checkAuthHeaders(w http.ResponseWriter, r *http.Request) (token, secret string, ok bool) {
	token = r.Header.Get("X-API-Token")
	secret = r.Header.Get("X-API-Secret")
	ok = token != secret && token != ""
	if !ok {
		writeJsonE(w, r, 403, wunderdns.ReturnError("X-* headers are missing"))
	}
	return
}

func signRequest(request *wunderdns.WunderRequest, token, secret string) {
	request.Auth = new(wunderdns.AuthHeader)
	request.Auth.Token = token
	t := time.Now().Unix()
	t -= t % 30
	s := new(strings.Builder)
	if request.Domain != nil {
		s.WriteString(string(request.Domain.View))
		s.WriteString(request.Domain.Name)
	}
	s.WriteString(string(request.Cmd))
	if request.Record != nil {
		for _, n := range request.Record {
			s.WriteString(string(n.Type))
			s.WriteString(n.Name)
			s.WriteString(strings.Join(n.Data, "@"))
		}
	}
	s.WriteString(fmt.Sprintf("%d", t))
	x := crypto.SHA256.New()
	x.Write([]byte(s.String()))
	v1 := fmt.Sprintf("%0x", x.Sum(nil))
	x.Reset()
	x.Write([]byte(fmt.Sprintf("%s@%s", secret, v1)))
	request.Auth.Sum = fmt.Sprintf("%0x", x.Sum(nil))
}

func mergeReply(reply ...*wunderdns.WunderReply) *wunderdns.WunderReply {
	ret := new(wunderdns.WunderReply)
	ret.Data = make(map[string]interface{})
	retmap := ret.Data.(map[string]interface{})
	for _, r := range reply {
		if ret.Status == "" {
			ret.Status = r.Status
		} else {
			if ret.Status != r.Status { // mismatch
				ret.Status = "MERGED"
			}
		}
		if m, ok := r.Data.(map[string]interface{}); ok {
			for k, v := range m {
				if _, ok := retmap[k]; !ok { // not exists ( normal )
					retmap[k] = v
				} else {
					newA := make([]interface{}, 0)
					newA = append(newA, retmap[k])
					newA = append(newA, v)
					retmap[k] = newA
				}
			}
		}
	}
	return ret
}

func signAndPush(request *wunderdns.WunderRequest, token, secret string) *wunderdns.WunderReply {
	if request.Domain != nil {
		if request.Domain.View == wunderdns.DomainViewAny {
			request.Domain.View = wunderdns.DomainViewPrivate
			signRequest(request, token, secret)
			re1 := producer.pushMessage(request)
			request.Domain.View = wunderdns.DomainViewPublic
			signRequest(request, token, secret)
			re2 := producer.pushMessage(request)
			return mergeReply(re1, re2)
		}
	}
	signRequest(request, token, secret)
	return producer.pushMessage(request)
}
