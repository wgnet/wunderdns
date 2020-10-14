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
	"net/http"
	"github.com/wgnet/wunderdns/wunderdns"
	"strconv"
)

func any2int(v interface{}) int {
	switch v.(type) {
	case int:
		return v.(int)
	case float32:
		return int(v.(float32))
	case float64:
		return int(v.(float64))
	case string:
		if i, e := strconv.Atoi(v.(string)); e == nil {
			return i
		}
	case bool:
		if v.(bool) {
			return 1
		}
	}
	return 0
}

func record2record(record map[string]interface{}) []*wunderdns.Record {
	ret := make([]*wunderdns.Record, 0)
	one := &wunderdns.Record{}
	one.Name = record["target"].(string)
	if ttl, ok := record["ttl"]; ok {
		one.TTL = any2int(ttl)
	} else {
		one.TTL = 0
	}
	one.Type = wunderdns.RecordType(record["type"].(string))
	one.Data = make([]string, 0)
	if data, ok := record["data"]; ok {
		switch data.(type) {
		case []string:
			one.Data = append(one.Data, data.([]string)...)
			break
		case string:
			one.Data = append(one.Data, data.(string))
			break
		}
		ret = append(ret, one)
	}
	return ret
}

func records2records(records []interface{}) []map[string]interface{} {
	ret := make([]map[string]interface{}, 0)
	for _, r := range records {
		switch r.(type) {
		case map[string]interface{}:
			ret = append(ret, r.(map[string]interface{}))
		default:
			panic(errors.New(fmt.Sprintf("invalid element type: %T", r)))
		}
	}
	return ret
}

func getDomainView(r *http.Request) wunderdns.DomainView {
	domainView := wunderdns.DomainView(r.FormValue("view"))
	if domainView == "" {
		domainView = wunderdns.DomainView(r.FormValue("type"))
		if domainView == "" {
			domainView = wunderdns.DomainViewAny
		}
	}
	switch domainView {
	case wunderdns.DomainViewPrivate, wunderdns.DomainViewPublic:
		return domainView
	default:
		return wunderdns.DomainViewAny
	}
}
