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
package wunderdns

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type Command string
type DomainView string
type RecordType string

type RecordsType []*Record

const (
	CommandCreateDomain  Command = "create_domain"
	CommandCreateRecord  Command = "create_record"
	CommandDeleteRecord  Command = "delete_record"
	CommandReplaceRecord Command = "replace_record"
	CommandListRecords   Command = "list_records"
	CommandListOwn       Command = "list_own"
	CommandListDomains   Command = "list_domains"
	CommandSearchRecord  Command = "search_record"
	CommandReplaceOwner  Command = "replace_owner"
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
	CommandSearchRecord:  true,
	CommandReplaceOwner:  true,
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
	Auth     *AuthHeader `json:"a"`
	Cmd      Command     `json:"c"`
	Domain   *Domain     `json:"d"`
	Record   []*Record   `json:"r"`
	NewToken string      `json:"n"`
	Pretty   bool        `json:"p"`
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
	isVault     bool
}

var authDataLock = sync.RWMutex{}

type Permission struct {
	Domain    Domain
	Permitted []Command
}

type VaultData struct {
	Enabled bool
	URL     string
	Token   string
	TTL     time.Duration
}
type Config struct {
	AMQPConfigs []*AMQPConfig
	PSQLConfigs []*PSQLConfig
	Auth        *AuthDatabase
	Vault       *VaultData
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

func (r *WunderRequest) toString() string {
	return fmt.Sprintf("auth: %s; command: %s; domain %s; record: %s",
		r.Auth.toString(),
		r.Cmd,
		r.Domain.toString(),
		RecordsType(r.Record).toString(),
	)
}

func (r *AuthHeader) toString() string {
	if r == nil {
		return "[nil]"
	}
	return fmt.Sprintf("[token:%s]", r.Token)
}

func (d *Domain) toString() string {
	if d == nil {
		return "[nil]"
	}
	return fmt.Sprintf("[view:%s/name:%s]", d.View, d.Name)
}

func (r RecordsType) toString() string {
	if r == nil {
		return "[nil]"
	}
	ret := make([]string, 0)
	for _, rec := range r {
		if rec == nil {
			ret = append(ret, "[nil]")
		} else {
			ret = append(ret, rec.toString())
		}

	}
	return fmt.Sprintf("[%s]", strings.Join(ret, ","))
}

func (r *Record) toString() string {
	return fmt.Sprintf("[name:%s/type:%s/data:[%s]/ttl:%d]", r.Name, r.Type, strings.Join(r.Data, ";"), r.TTL)
}
