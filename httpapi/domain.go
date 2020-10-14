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
	"log"
	"net/http"
	"github.com/wgnet/wunderdns/wunderdns"
)

func apiDomainFunc(w http.ResponseWriter, r *http.Request) {
	pretty := false
	if _, ok := r.URL.Query()["pretty"]; ok {
		pretty = true
	}
	if token, secret, ok := checkAuthHeaders(w, r); !ok {
		return
	} else {
		switch r.Method {
		case http.MethodPost, http.MethodPut:
			dec := json.NewDecoder(r.Body)
			req := make(map[string]interface{})
			if e := dec.Decode(&req); e != nil {
				log.Print("Error decoding json: ", e.Error())
				writeJsonE(w, r, 503, "Internal Server Error")
				return
			}
			writeJson(w, r, apiCreateDomain(req, token, secret))
		case http.MethodGet:
			writeJson(w, r, apiListDomains(getDomainView(r), token, secret, pretty))
		default:
			writeJsonE(w, r, 422, "Method not supported")
		}
	}
}

func apiCreateDomain(params map[string]interface{}, token, secret string) *wunderdns.WunderReply {
	defer func() {
		if e := recover(); e != nil {
			log.Printf("apiCreateDomain(%v) error: %v", params, e)
		}
	}()
	req := &wunderdns.WunderRequest{
		Domain: &wunderdns.Domain{
			Name: params["domain"].(string),
			View: wunderdns.DomainViewAny,
		},
		Cmd: wunderdns.CommandCreateDomain,
	}
	return signAndPush(req, token, secret)
}

func apiListDomains(domainView wunderdns.DomainView, token, secret string, pretty bool) *wunderdns.WunderReply {
	req := &wunderdns.WunderRequest{
		Domain: &wunderdns.Domain{
			Name: "*",
			View: domainView,
		},
		Cmd:    wunderdns.CommandListDomains,
		Pretty: pretty,
	}
	return signAndPush(req, token, secret)
}
