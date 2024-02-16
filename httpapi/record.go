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
	"fmt"
	"github.com/asaskevich/govalidator"
	"github.com/wgnet/wunderdns/wunderdns"
	"log"
	"net/http"
)

func apiRecordFunc(w http.ResponseWriter, r *http.Request) {
	pretty := false
	if _, ok := r.URL.Query()["pretty"]; ok {
		pretty = true
	}
	if token, secret, ok := checkAuthHeaders(w, r); !ok {
		return
	} else {
		switch r.Method {
		case http.MethodGet:
			if _, ok := r.URL.Query()["search"]; ok {
				if record := r.FormValue("record"); record != "" {
					writeJson(w, r, apiSearchRecords(token, secret, r.FormValue("record"), pretty))
				} else {
					writeJsonE(w, r, 422, "record parameter is missing")
				}
				return
			}
			if _, ok := r.URL.Query()["own"]; ok {
				writeJson(w, r, apiListOwnRecords(token, secret, pretty))
				return
			}
			if domain := r.FormValue("domain"); govalidator.IsDNSName(domain) {
				record := r.FormValue("record")
				writeJson(w, r, apiListRecords(domain, getDomainView(r), record, token, secret, pretty))
			} else {
				writeJsonE(w, r, 422, "Domain parameter is missing")
			}
		case http.MethodPut, http.MethodDelete, http.MethodPost:
			dec := json.NewDecoder(r.Body)
			req := make(map[string]interface{})
			if e := dec.Decode(&req); e != nil {
				log.Print("Error decoding json: ", e.Error())
				writeJsonE(w, r, 503, "Internal Server Error")
				return
			}
			if r.Method == http.MethodPost {
				writeJson(w, r, apiCreateRecord(req, token, secret))
			} else if r.Method == http.MethodDelete {
				writeJson(w, r, apiDeleteRecord(req, token, secret))
			} else {
				writeJson(w, r, apiReplaceRecord(req, token, secret))
			}
		default:
			writeJsonE(w, r, 422, "Method not supported")
		}
	}
}

func apiCreateRecord(req map[string]interface{}, token, secret string) (r *wunderdns.WunderReply) {
	defer func() {
		if e := recover(); e != nil {
			log.Printf("apiCreateRecord(%v) error: %v", req, e)
			r = wunderdns.ReturnError(e)
		}
	}()
	errors := 0
	success := 0
	ret := map[string]interface{}{
		"success": &success,
		"error":   &errors,
		"replies": make([]interface{}, 0),
	}
	for _, x := range []string{"domain", "record"} {
		if _, ok := req[x]; !ok {
			return wunderdns.ReturnError("JSON field missing: ", x)
		}
	}

	records := make([]map[string]interface{}, 0)
	switch req["record"].(type) {
	case map[string]interface{}:
		records = append(records, req["record"].(map[string]interface{}))
	case []interface{}:
		records = append(records, records2records(req["record"].([]interface{}))...)
	default:
		return wunderdns.ReturnError("Invalid record field type: must be a Hash or Array[Hash], got ",
			fmt.Sprintf("%T", req["record"]))
	}
	for _, r := range records {
		if !checkRecord(r) {
			return wunderdns.ReturnError("Invalid record(s)")
		}
	}
	for _, r := range records {
		wreq := &wunderdns.WunderRequest{
			Domain: &wunderdns.Domain{},
		}
		wreq.Cmd = wunderdns.CommandCreateRecord
		wreq.Domain.Name = req["domain"].(string)
		wreq.Domain.View = wunderdns.DomainView(r["view"].(string))
		wreq.Record = record2record(r)
		signRequest(wreq, token, secret)
		reply := producer.pushMessage(wreq)
		ret["replies"] = append(ret["replies"].([]interface{}), reply.Data)
		if reply.Status == "ERROR" {
			errors++
		} else {
			success++
		}
	}
	var status string
	if errors == 0 {
		status = "SUCCESS"
	} else {
		status = "ERROR"
	}
	return &wunderdns.WunderReply{
		Status: status,
		Data:   ret,
	}
}

func apiReplaceRecord(req map[string]interface{}, token, secret string) (r *wunderdns.WunderReply) {
	defer func() {
		if e := recover(); e != nil {
			log.Printf("apiReplaceRecord(%v) error: %v", req, e)
			r = wunderdns.ReturnError(e)
		}
	}()
	errors := 0
	success := 0
	ret := map[string]interface{}{
		"success": &success,
		"error":   &errors,
		"replies": make([]interface{}, 0),
	}
	for _, x := range []string{"domain", "record"} {
		if _, ok := req[x]; !ok {
			return wunderdns.ReturnError("JSON field missing: ", x)
		}
	}

	records := make([]map[string]interface{}, 0)
	switch req["record"].(type) {
	case map[string]interface{}:
		records = append(records, req["record"].(map[string]interface{}))
	case []interface{}:
		records = append(records, records2records(req["record"].([]interface{}))...)
	default:
		return wunderdns.ReturnError("Invalid record field type: must be a Hash or Array[Hash], got ",
			fmt.Sprintf("%T", req["record"]))

	}

	for _, r := range records {
		if !checkRecord(r) {
			return wunderdns.ReturnError("Invalid record(s)")
		}
	}
	for _, r := range records {
		wreq := &wunderdns.WunderRequest{
			Domain: &wunderdns.Domain{},
		}
		wreq.Cmd = wunderdns.CommandReplaceRecord
		wreq.Domain.Name = req["domain"].(string)
		wreq.Domain.View = wunderdns.DomainView(r["view"].(string))
		wreq.Record = record2record(r)
		signRequest(wreq, token, secret)
		reply := producer.pushMessage(wreq)
		ret["replies"] = append(ret["replies"].([]interface{}), reply.Data)
		if reply.Status == "ERROR" {
			errors++
		} else {
			success++
		}
	}
	var status string
	if errors == 0 {
		status = "SUCCESS"
	} else {
		status = "ERROR"
	}
	return &wunderdns.WunderReply{
		Status: status,
		Data:   ret,
	}
}

func apiDeleteRecord(req map[string]interface{}, token, secret string) (r *wunderdns.WunderReply) {
	defer func() {
		if e := recover(); e != nil {
			log.Printf("apiDeleteRecord(%v) error: %v", req, e)
			r = wunderdns.ReturnError(e)
		}
	}()
	errors := 0
	success := 0
	ret := map[string]interface{}{
		"success": &success,
		"error":   &errors,
		"replies": make([]interface{}, 0),
	}
	for _, x := range []string{"domain", "record"} {
		if _, ok := req[x]; !ok {
			return wunderdns.ReturnError("JSON field missing: ", x)
		}
	}

	records := make([]map[string]interface{}, 0)
	switch req["record"].(type) {
	case map[string]interface{}:
		records = append(records, req["record"].(map[string]interface{}))
	case []interface{}:
		records = append(records, records2records(req["record"].([]interface{}))...)
	default:
		return wunderdns.ReturnError("Invalid record field type: must be a Hash or Array[Hash], got ",
			fmt.Sprintf("%T", req["record"]))

	}

	for _, r := range records {
		if !checkRecord(r) {
			return wunderdns.ReturnError("Invalid record(s)")
		}
	}
	for _, r := range records {
		wreq := &wunderdns.WunderRequest{
			Domain: &wunderdns.Domain{},
		}
		wreq.Cmd = wunderdns.CommandDeleteRecord
		wreq.Domain.Name = req["domain"].(string)
		wreq.Domain.View = wunderdns.DomainView(r["view"].(string))
		wreq.Record = record2record(r)
		signRequest(wreq, token, secret)
		reply := producer.pushMessage(wreq)
		ret["replies"] = append(ret["replies"].([]interface{}), reply.Data)
		if reply.Status == "ERROR" {
			errors++
		} else {
			success++
		}
	}
	var status string
	if errors == 0 {
		status = "SUCCESS"
	} else {
		status = "ERROR"
	}
	return &wunderdns.WunderReply{
		Status: status,
		Data:   ret,
	}
}

func checkRecord(rec map[string]interface{}) bool {
	for _, x := range []string{"target", "type", "view", "data"} {
		if _, ok := rec[x]; !ok {
			return false
		}
	}
	return true
}

func apiListRecords(domain string, domainView wunderdns.DomainView, recordName string, token, secret string, pretty bool) *wunderdns.WunderReply {
	req := &wunderdns.WunderRequest{
		Cmd: wunderdns.CommandListRecords,
		Domain: &wunderdns.Domain{
			Name: domain,
			View: domainView,
		},
		Pretty: pretty,
	}
	if recordName != "" {
		req.Record = []*wunderdns.Record{{
			Name: recordName,
		}}
	}
	return signAndPush(req, token, secret)
}

func apiListOwnRecords(token, secret string, pretty bool) *wunderdns.WunderReply {
	req := &wunderdns.WunderRequest{
		Cmd: wunderdns.CommandListOwn,
		Domain: &wunderdns.Domain{
			Name: wunderdns.DomainNameAny,
			View: wunderdns.DomainViewAny,
		},
		Pretty: pretty,
	}
	return signAndPush(req, token, secret)
}
func apiSearchRecords(token, secret string, record string, pretty bool) *wunderdns.WunderReply {
	req := &wunderdns.WunderRequest{
		Cmd: wunderdns.CommandSearchRecord,
		Domain: &wunderdns.Domain{
			Name: "",
			View: wunderdns.DomainViewAny,
		},
		Record: []*wunderdns.Record{
			&wunderdns.Record{
				Name: record,
			},
		},
		Pretty: pretty,
	}
	return signAndPush(req, token, secret)
}
