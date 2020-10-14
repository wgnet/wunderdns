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

import "testing"

var authdb = &AuthDatabase{
	"test": {
		Token:  "test",
		Secret: "test",
		Permissions: []Permission{
			{
				Domain: Domain{
					"test.com",
					DomainViewPublic,
				},
				Permitted: []Command{
					CommandCreateRecord, CommandListRecords,
				},
			},
		},
	},
	"test2": {
		Token:  "test2",
		Secret: "test2",
		Permissions: []Permission{
			{
				Domain: Domain{
					Name: DomainNameAny,
					View: DomainViewAny,
				},
				Permitted: []Command{
					CommandListDomains, CommandListRecords,
				},
			},
		},
		Priority: 0,
	},
	"test3": {
		Token:  "test3",
		Secret: "test3",
		Permissions: []Permission{
			{
				Domain: Domain{
					Name: "*1.test.com",
					View: DomainViewAny,
				},
				Permitted: []Command{
					CommandListDomains, CommandListRecords,
				},
			},
		},
		Priority: 0,
	},
}

func TestIsPermitted(t *testing.T) {
	testCases := []*WunderRequest{
		{
			Auth: &AuthHeader{
				Token:    "test",
				Sum:      "x",
				priority: 0,
			},
			Cmd: CommandCreateRecord,
			Domain: &Domain{
				Name: "test.com",
				View: DomainViewPublic,
			},
			Record: nil,
			Pretty: false,
		},
		{
			Auth: &AuthHeader{
				Token:    "test",
				Sum:      "x",
				priority: 0,
			},
			Cmd: CommandCreateRecord,
			Domain: &Domain{
				Name: "test.com",
				View: DomainViewAny,
			},
			Record: nil,
			Pretty: false,
		},
		{
			Auth: &AuthHeader{
				Token:    "test",
				Sum:      "x",
				priority: 0,
			},
			Cmd: CommandDeleteRecord,
			Domain: &Domain{
				Name: "test.com",
				View: DomainViewPublic,
			},
			Record: nil,
			Pretty: false,
		},
		{
			Auth: &AuthHeader{
				Token:    "test2",
				Sum:      "x",
				priority: 0,
			},
			Cmd: CommandListDomains,
			Domain: &Domain{
				Name: "test.com",
				View: DomainViewPublic,
			},
			Record: nil,
			Pretty: false,
		},
		{
			Auth: &AuthHeader{
				Token:    "test2",
				Sum:      "x",
				priority: 0,
			},
			Cmd: CommandListRecords,
			Domain: &Domain{
				Name: "test.com",
				View: DomainViewPublic,
			},
			Record: nil,
			Pretty: false,
		},
		{
			Auth: &AuthHeader{
				Token:    "test2",
				Sum:      "x",
				priority: 0,
			},
			Cmd: CommandCreateRecord,
			Domain: &Domain{
				Name: "test.com",
				View: DomainViewPublic,
			},
			Record: nil,
			Pretty: false,
		},
		{
			Auth: &AuthHeader{
				Token:    "test3",
				Sum:      "x",
				priority: 0,
			},
			Cmd: CommandListRecords,
			Domain: &Domain{
				Name: "31.test.com",
				View: DomainViewPublic,
			},
			Record: nil,
			Pretty: false,
		},
	}
	expected := []bool{
		true, false, false, true, true, false, true,
	}
	for i := 0; i < len(expected); i++ {
		if authdb.isPermitted(testCases[i]) != expected[i] {
			t.Errorf("case number %d doesn't match result", i+1)
		}
	}

}
