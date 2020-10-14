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
	"strings"
	"testing"
)

func TestRFCRequest(t *testing.T) {
	records := []Record{
		{
			Name: "*.test1",
			Type: "NONEXISTS",
			Data: nil,
			TTL:  0,
			view: "private",
		},
		{
			Name: "test1.*",
			Type: "",
			Data: nil,
			TTL:  0,
			view: "",
		},
	}
	results := []bool{true,false}
	for i,v := range records {
		req := &WunderRequest{
			Domain: &Domain{
				Name: "example.com",
				View:DomainViewPrivate,
			},
			Record: []*Record{&v},
		}
		e := checkRFCRequest(req)
		if (e == nil) != results[i] {
			t.Error("Test ", i, "failed with error ", e.Error())
		}
	}
}
func TestCheckRecordTypeA(t *testing.T) {

	values := []*Record{
		{
			Name: "",
			Type: "A",
			Data: []string{},
			TTL:  0,
			view: "", // invalid
		},
		{
			Name: "",
			Type: "A",
			Data: []string{"192.168.0.1"}, // valid
			TTL:  0,
			view: "",
		},
		{
			Name: "",
			Type: "A",
			Data: []string{"192.168.256.1"}, // invalid
			TTL:  0,
			view: "",
		},
	}
	for i, r := range values {
		e := checkRecordTypeA(r)
		if (e == nil && i != 1) || (i == 1 && e != nil) {
			t.Errorf("%s checker failed on case #%d, %v", r.Type, i, r)
		}
	}
}
func TestCheckRecordTypeAAAA(t *testing.T) {
	values := []*Record{
		{
			Name: "",
			Type: "AAAA",
			Data: []string{},
			TTL:  0,
			view: "", // invalid
		},
		{
			Name: "",
			Type: "AAAA",
			Data: []string{"::1"}, // valid
			TTL:  0,
			view: "",
		},
		{
			Name: "",
			Type: "AAAA",
			Data: []string{"fe80::50a3:ddcb:6e94:3aax"}, // invalid
			TTL:  0,
			view: "",
		},
		{
			Name: "",
			Type: "AAAA",
			Data: []string{"192.168.0.1"}, // invalid
			TTL:  0,
			view: "",
		},
	}
	for i, r := range values {
		e := checkRecordTypeAAAA(r)
		if (e == nil && i != 1) || (i == 1 && e != nil) {
			t.Errorf("%s checker failed on case #%d, %v", r.Type, i, r)
		}
	}
}
func TestCheckRecordTypeCNAME(t *testing.T) {
	values := []*Record{
		{
			Name: "a",
			Type: "CNAME",
			Data: []string{},
			TTL:  0,
			view: "", // invalid
		},
		{
			Name: "a",
			Type: "CNAME",
			Data: []string{"test.com"}, // valid
			TTL:  0,
			view: "",
		},
		{
			Name: "",
			Type: "CNAME",
			Data: []string{"test.com"}, // invalid
			TTL:  0,
			view: "",
		},
		{
			Name: "a",
			Type: "CNAME",
			Data: []string{"192.168.0.1"}, // invalid
			TTL:  0,
			view: "",
		},
		{
			Name: "a",
			Type: "CNAME",
			Data: []string{"---aaccaa.com"}, // invalid
			TTL:  0,
			view: "",
		},
		{
			Name: "a",
			Type: "CNAME",
			Data: []string{"test.com", "test2.com"}, // invalid
			TTL:  0,
			view: "",
		},
	}
	for i, r := range values {
		e := checkRecordTypeCNAME(r)
		if (e == nil && i != 1) || (i == 1 && e != nil) {
			t.Errorf("%s checker failed on case #%d, %v", r.Type, i, r)
		}
	}
}
func TestCheckRecordTypeTXT(t *testing.T) {
	values := []*Record{
		{
			Name: "",
			Type: "TXT",
			Data: []string{},
			TTL:  0,
			view: "", // invalid
		},
		{
			Name: "",
			Type: "TXT",
			Data: []string{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, // valid
			TTL:  0,
			view: "",
		},
		{
			Name: "",
			Type: "TXT",
			Data: []string{strings.Repeat("a", 256)}, // invalid
			TTL:  0,
			view: "",
		},
	}
	for i, r := range values {
		e := checkRecordTypeTXT(r)
		if (e == nil && i != 1) || (i == 1 && e != nil) {
			t.Errorf("%s checker failed on case #%d, %v", r.Type, i, r)
		}
	}
}

func TestCheckRecordTypeSRV(t *testing.T) {
	// TODO
}
func TestCheckRecordTypeMX(t *testing.T) {
	// TODO
}
func TestCheckRecordTypeNS(t *testing.T) {
	// TODO
}
func TestCheckRecordTypePTR(t *testing.T) {
	// TODO
}
func TestCheckRecordTypeSOA(t *testing.T) {
	currentSerial, _ := generateNewSerial("0")
	startSOA := fmt.Sprintf("ns1.wargaming.net admins.wargaming.net %s", currentSerial)
	values := []*Record{
		{
			Name: "test1", // invalid
			Type: "SOA",
			Data: []string{},
			TTL:  0,
			view: "",
		},
		{
			Name: "", // valid
			Type: "SOA",
			Data: []string{fmt.Sprintf("%s %s", startSOA, "900 600 86400 600")},
			TTL:  0,
			view: "",
		},
		{
			Name: "", // invalid
			Type: "SOA",
			Data: []string{fmt.Sprintf("%s %s", startSOA, "900 600 86400 600"), "aaa"},
			TTL:  0,
			view: "",
		},
		{
			Name: "", // retry
			Type: "SOA",
			Data: []string{fmt.Sprintf("%s %s", startSOA, "600 900 86400 600")},
			TTL:  0,
			view: "",
		},
		{
			Name: "", // expire
			Type: "SOA",
			Data: []string{fmt.Sprintf("%s %s", startSOA, "600 900 1400 600")},
			TTL:  0,
			view: "",
		},
		{
			Name: "", // ttl
			Type: "SOA",
			Data: []string{fmt.Sprintf("%s %s", startSOA, "600 900 86400 6000000")},
			TTL:  0,
			view: "",
		},
		{
			Name: "", // format
			Type: "SOA",
			Data: []string{"a b c d ee"},
			TTL:  0,
			view: "",
		},
		{
			Name: "",
			Type: "SOA", // invalid dns
			Data: []string{"ns66.wargaming.net admins.wargaming.net 2020020403 900 600 86400 600"},
			TTL:  0,
			view: "",
		},
	}
	for i, r := range values {
		e := checkRecordTypeSOA(r)
		if (e == nil && i != 1) || (i == 1 && e != nil) {
			t.Errorf("%s checker failed on case #%d, %v", r.Type, i, r)
		}
	}
}
