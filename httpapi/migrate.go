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
package httpapi

import (
	"encoding/json"
	"fmt"
	"github.com/wgnet/wunderdns/wunderdns"
	"log"
	"net/http"
)

func apiMigrateFunc(w http.ResponseWriter, r *http.Request) {
	if token, secret, ok := checkAuthHeaders(w, r); !ok {
		return
	} else {
		switch r.Method {
		case http.MethodPut:
			dec := json.NewDecoder(r.Body)
			req := make(map[string]interface{})
			if e := dec.Decode(&req); e != nil {
				log.Print("Error decoding json: ", e.Error())
				writeJsonE(w, r, 503, "json decoding error")
				return
			}
			/** validation part **/
			for _, field := range []string{"new_token", "domain", "domain_view", "records"} {
				if _, ok := req[field]; !ok {
					writeJsonE(w, r, 422, fmt.Sprintf("field %s now found in the request", field))
					return
				}
				_test := req[field]
				if field != "records" {
					if _, ok := _test.(string); !ok {
						writeJsonE(w, r, 422, fmt.Sprintf("field %s is not a string", field))
						return
					}
				} else {
					if _, ok := _test.([]interface{}); !ok {
						writeJsonE(w, r, 422, fmt.Sprintf("field %s is not an array", field))
						return
					}
				}
			}
			switch req["domain_view"] {
			case "public", "private", "*":
				writeJson(w, r, apiReplaceOwner(token, secret,
					req["domain"].(string),
					req["domain_view"].(string),
					req["new_token"].(string),
					records2records(req["records"].([]interface{}))))
			default:
				writeJsonE(w, r, 422, "domain_view is not in (public,private,*)")
				return
			}
		default:
			writeJsonE(w, r, 501, "not implemented")

		}
	}
}

func apiReplaceOwner(token, secret string, domain string, domainView string, newToken string, records []map[string]interface{}) *wunderdns.WunderReply {
	recs := make([]*wunderdns.Record, 0)
	for _, req := range records {
		req["data"] = ""
		recs = append(recs, record2record(req)...)
	}
	for i, _ := range recs {
		recs[i].Data = []string{}
	}
	req := &wunderdns.WunderRequest{
		Cmd: wunderdns.CommandReplaceOwner,
		Domain: &wunderdns.Domain{
			Name: domain,
			View: wunderdns.DomainView(domainView),
		},
		NewToken: newToken,
		Record:   recs,
		Pretty:   false,
	}
	return signAndPush(req, token, secret)
}
