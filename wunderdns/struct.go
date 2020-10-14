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

type Command string
type DomainView string
type RecordType string

const (
	CommandCreateDomain  Command = "create_domain"
	CommandCreateRecord  Command = "create_record"
	CommandDeleteRecord  Command = "delete_record"
	CommandReplaceRecord Command = "replace_record"
	CommandListRecords   Command = "list_records"
	CommandListOwn       Command = "list_own"
	CommandListDomains   Command = "list_domains"
	CommandAny           Command = "*"
)

const (
	DomainViewPublic  DomainView = "public"
	DomainViewPrivate DomainView = "private"
	DomainViewAny     DomainView = "*"
)

const (
	RecordTypeA     RecordType = "A"
	RecordTypeAAAA  RecordType = "AAAA"
	RecordTypeCNAME RecordType = "CNAME"
	RecordTypeTXT   RecordType = "TXT"
	RecordTypeSRV   RecordType = "SRV"
	RecordTypeMX    RecordType = "MX"
	RecordTypeNS    RecordType = "NS"
	RecordTypePTR   RecordType = "PTR"
	RecordTypeSOA   RecordType = "SOA"
)

var domainViews = map[DomainView]bool{
	DomainViewPrivate: true,
	DomainViewPublic:  true,
	DomainViewAny:     true,
}

var commands = map[Command]bool{
	CommandCreateRecord:  true,
	CommandListRecords:   true,
	CommandAny:           true,
	CommandDeleteRecord:  true,
	CommandCreateDomain:  true,
	CommandListDomains:   true,
	CommandReplaceRecord: true,
}

var recordTypes = map[RecordType]bool{
	RecordTypeA:     true,
	RecordTypeAAAA:  true,
	RecordTypeCNAME: true,
	RecordTypeTXT:   true,
	RecordTypeSRV:   true,
	RecordTypeMX:    true,
	RecordTypeNS:    true,
	RecordTypeSOA:   true,
	RecordTypePTR:   true,
}

const DomainNameAny string = "*"

type WunderRequest struct {
	Auth   *AuthHeader `json:"a"`
	Cmd    Command     `json:"c"`
	Domain *Domain     `json:"d"`
	Record []*Record   `json:"r"`
	Pretty bool        `json:"p"`
}

type WunderReply struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data"`
}

type AuthHeader struct {
	Token    string `json:"t"`
	Sum      string `json:"x"`
	priority int
}

type Domain struct {
	Name string     `json:"n"`
	View DomainView `json:"v"`
}

type DomainPretty struct {
	Name string     `json:"name"`
	View DomainView `json:"type"`
}

type Record struct {
	Name string     `json:"n"`
	Type RecordType `json:"t"`
	Data []string   `json:"d"`
	TTL  int        `json:"l"`
	view DomainView
}

type RecordPretty struct {
	Name string     `json:"name"`
	Type RecordType `json:"type"`
	Data []string   `json:"data"`
	TTL  int        `json:"ttl"`
	view DomainView
}

type AuthDatabase map[string]AuthData

type AuthData struct {
	Token       string
	Secret      string
	Permissions []Permission
	Priority    int
}

type Permission struct {
	Domain    Domain
	Permitted []Command
}

type Config struct {
	AMQPConfigs []*AMQPConfig
	PSQLConfigs []*PSQLConfig
	Auth        *AuthDatabase
}

type AMQPConfig struct {
	URL      string
	Exchange string
}

type PSQLConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
	SSL      bool
	View     DomainView
}
