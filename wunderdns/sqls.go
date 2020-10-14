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
	"database/sql"
	"errors"
	"fmt"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"strconv"
	"strings"
)

type database struct {
	config *PSQLConfig
	//lock        *sync.Mutex
	connection  *sql.DB
	transaction *sql.Tx
}

var databases = make([]*database, 0)

func initSQLs() {
	for _, c := range globalConfig.PSQLConfigs {
		databases = append(databases, &database{
			config: c,
			//lock:        new(sync.Mutex),
			connection:  nil,
			transaction: nil,
		})
		logging.Info("Found database: ", c.Host)
	}
}

func applyCommandMerge(request *WunderRequest) (re map[DomainView][]interface{}, e error) {
	defer allRollback(request.Domain.View)
	if e = allBegin(request.Domain.View); e != nil {
		logging.Fatal("AllBegin ERROR: ", e.Error())
		return
	}
	re = make(map[DomainView][]interface{})
	for _, d := range databases {
		if d.config.View != request.Domain.View && request.Domain.View != DomainViewAny {
			continue // skip
		}
		var data []interface{}
		data, e = d.applyCommandData(request)
		if e != nil {
			allRollback(request.Domain.View)
			return
		}
		if _, ok := re[d.config.View]; !ok {
			re[d.config.View] = make([]interface{}, 0)
		}
		re[d.config.View] = append(re[d.config.View], data...)
	}
	if e = allCommit(request.Domain.View); e != nil {
		logging.Fatal("AllCommit ERROR: ", e.Error())
	}
	return
}
func applyCommand(request *WunderRequest) (n int, e error) {
	defer allRollback(request.Domain.View)
	if e = allBegin(request.Domain.View); e != nil {
		logging.Fatal("AllBegin ERROR: ", e.Error())
		return
	}
	n = 0
	for _, d := range databases {
		if d.config.View != request.Domain.View && request.Domain.View != DomainViewAny {
			continue // skip
		}
		var n_ int
		n_, e = d.applyCommand(request)
		if e != nil {
			allRollback(request.Domain.View)
			return
		}
		n += n_
	}
	if e = allCommit(request.Domain.View); e != nil {
		logging.Fatal("AllCommit ERROR: ", e.Error())
	}
	return
}

func allBegin(t DomainView) error {
	for _, d := range databases {
		if d.config.View == t || d.config.View == DomainViewAny || t == DomainViewAny {
			if e := d.begin(); e != nil {
				return e
			}
		}
	}
	return nil
}

func allCommit(t DomainView) error {

	for _, d := range databases {
		if d.config.View == t || d.config.View == DomainViewAny || t == DomainViewAny {
			if e := d.commit(); e != nil {
				return e
			}
		}
	}
	return nil
}

func allRollback(t DomainView) {
	for _, d := range databases {
		if d.config.View == t || d.config.View == DomainViewAny || t == DomainViewAny {
			d.rollback()
		}
	}
}

func (d *database) begin() error {
	//d.lock.Lock()
	needConnect := false
	if d.connection == nil {
		needConnect = true
	} else if e := d.connection.Ping(); e != nil {
		d.connection.Close()
		d.connection = nil
		needConnect = true
	}
	if needConnect {
		if c, e := sql.Open("postgres", d.config.connString()); e != nil {
			return e
		} else {
			d.connection = c
		}
	}
	// close tx if any
	if d.transaction != nil {
		d.transaction.Rollback()
		d.transaction = nil
	}
	if t, e := d.connection.Begin(); e != nil {
		return e
	} else {
		d.transaction = t
	}
	return nil
}

func (d *database) applyCommand(request *WunderRequest) (n int, e error) {
	switch request.Cmd {

	case CommandCreateDomain:
		if n, e = d.sqlInsertDomain(request.Domain.Name); e != nil {
			return
		}
	case CommandCreateRecord:
		for _, r := range request.Record {
			if n, e = d.sqlInsertRecord(request.Domain.Name,
				request.Domain.record2dns(r),
				string(r.Type),
				r.TTL, r.Data, request.Auth.Token,
			); e != nil {
				return
			}
		}
		d.sqlUpdateSOA(request.Domain.Name)
	case CommandDeleteRecord:
		for _, r := range request.Record {
			if n, e = d.sqlDeleteRecord(request.Domain.Name,
				request.Domain.record2dns(r),
				string(r.Type),
				r.TTL, r.Data, request.Auth.Token); e != nil {
				return
			}
		}
		d.sqlUpdateSOA(request.Domain.Name)
	case CommandReplaceRecord:
		for _, r := range request.Record {
			if n, e = d.sqlReplaceRecord(request.Domain.Name,
				request.Domain.record2dns(r),
				string(r.Type),
				r.TTL, r.Data, request.Auth.Token); e != nil {
				return
			}
		}
		d.sqlUpdateSOA(request.Domain.Name)
	default:
		e = errors.New("not implemented yet")

	}
	return
}

func (d *database) applyCommandData(request *WunderRequest) (data []interface{}, e error) {
	switch request.Cmd {
	case CommandListDomains:
		data, e = d.getDomains(request.Pretty)
		return
	case CommandListRecords:
		data, e = d.sqlGetDomainRecords(request.Domain.Name, request.Record, request.Pretty)
		return

	case CommandListOwn:
		data, e = d.sqlGetOwnRecords(request.Auth.Token, request.Pretty)
		return
	default:
		e = errors.New("not implemented yet")
		return
	}
	return
}

func (d *database) commit() error {
	defer func() {
		d.transaction = nil
		//d.lock.Unlock()
	}()
	if d.transaction != nil {
		return d.transaction.Commit()
	} else {
		return errors.New("transaction is nill")
	}
}

func (d *database) rollback() error {
	defer func() {
		d.transaction = nil
		//d.lock.Unlock()
	}()
	if d.transaction != nil {
		return d.transaction.Rollback()
	} else {
		return errors.New("transaction is nill")
	}
}

func (d *database) sqlGetDomainId(domain string) (id int, e error) {
	e = nil
	id = -1
	if d.transaction == nil {
		e = d.begin()
		if e != nil {
			return
		}
	}
	e = d.transaction.QueryRow(`SELECT id FROM domains WHERE name=$1`,
		domain).Scan(&id)
	if e == sql.ErrNoRows {
		e = errors.New(fmt.Sprintf("domain %s not found", domain))
	}
	return
}

func (d *database) sqlUpdateSOA(domain string) (serial int, e error) {
	domainId, e := d.sqlGetDomainId(domain)
	if e != nil {
		return
	}
	var soa string
	var id int
	e = d.transaction.QueryRow(`SELECT id,content FROM records WHERE type=$1 AND domain_id=$2`,
		"SOA", domainId).Scan(&id, &soa)
	if e != nil {
		return
	}
	soa = strings.TrimSpace(soa)
	parts := strings.Split(soa, " ")
	parts[2], e = generateNewSerial(parts[2]) // update serial
	soa = strings.Join(parts, " ")
	_, e = d.transaction.Exec(`UPDATE records SET content=$1 WHERE id=$2`, soa, id)
	if e != nil {
		return 0, e
	}
	serial, e = strconv.Atoi(parts[2])
	logging.Info("Updating soa for ", domain, " to ", serial)
	return
}

func (d *database) sqlInsertRecord(domain, name, rtype string, ttl int, data []string, owner string) (n int, e error) {
	domainId, e := d.sqlGetDomainId(domain)
	if e != nil {
		return
	}
	// check for same record name / CNAME
	{

		var rows *sql.Rows
		rows, e = d.transaction.Query(`SELECT name,type FROM records WHERE domain_id=$1 and name=$2`,
			domainId, name)
		if e == sql.ErrNoRows {
			e = nil
		} else if e != nil {
			return
		} else {
			for rows.Next() {
				var eName, eType string
				rows.Scan(&eName, &eType)
				if strings.ToUpper(eType) == string(RecordTypeCNAME) {
					e = errors.New("(sql) multiple CNAME declaration")
					return
				}
				if strings.ToUpper(rtype) == string(RecordTypeCNAME) {
					e = errors.New("(json) multiple CNAME declaration")
					return
				}
				if strings.ToUpper(rtype) == strings.ToUpper(eType) && strings.ToUpper(eType) == string(RecordTypePTR) {
					e = errors.New("(sql) multiple PTR declaration")
					return
				}
				if strings.ToUpper(rtype) == strings.ToUpper(eType) && strings.ToUpper(eType) == string(RecordTypeSOA) {
					e = errors.New("(sql) multiple SOA declaration")
					return
				}
			}
		}
	}
	n = 0
	for _, elem := range data {
		prio := 0
		if rtype == string(RecordTypeMX) {
			mx := strings.Split(elem, " ")
			prio, _ = strconv.Atoi(mx[0])
			elem = mx[1]
		} else if rtype == string(RecordTypeSRV) {
			srv := strings.Split(elem, " ")
			prio, _ = strconv.Atoi(srv[0])
			elem = strings.Join(srv[1:], " ")
		}
		_, e = d.transaction.Exec(`INSERT INTO records_api(domain_id,name,type,content,ttl,prio,owner) 
VALUES($1,$2,$3,$4,$5,$6,$7)`,
			domainId,
			name,
			rtype,
			elem,
			ttl,
			prio,
			owner,
		)
		if e != nil {
			return
		} else {
			n += 1
		}
	}
	logging.Debug("Inserted ", len(data), " rows to ", domain, " ( ", name, "/", rtype, "/[", strings.Join(data, ", "), "] ) by ", owner)
	return
}

func (d *database) sqlDeleteRecord(domain, name, rtype string, ttl int, data []string, owner string) (n int, e error) {
	domainId, e := d.sqlGetDomainId(domain)
	if e != nil {
		return
	}
	delme := make([]int, 0)
	var rows *sql.Rows
	rows, e = d.transaction.Query(`SELECT id,content,prio,ttl,owner FROM records_api WHERE domain_id=$1 AND type=$2 AND name=$3`,
		domainId, rtype, name)
	if e != nil {
		return
	}
	for rows.Next() {
		var id int
		var content string
		var lprio int
		var lttl int
		var lowner string
		if e = rows.Scan(&id, &content, &lprio, &lttl, &lowner); e != nil {
			return
		}
		if lowner != owner {
			e = errors.New(fmt.Sprintf("sql: you're not an owner of record %s [%d/%s/%d/%s]", name, id, content, lttl, lowner))
			return
		}
		// TODO: skip ttl check at all
		//if ttl > 0 {
		//	if ttl != lttl {
		//		continue
		//	}
		//}
		if data == nil {
			delme = append(delme, id)
		} else {
			for _, element := range data {
				prio := 0
				if rtype == string(RecordTypeMX) {
					mx := strings.Split(element, " ")
					prio, _ = strconv.Atoi(mx[0])
					element = mx[1]
				} else if rtype == string(RecordTypeSRV) {
					srv := strings.Split(element, " ")
					prio, _ = strconv.Atoi(srv[0])
					element = strings.Join(srv[1:], " ")
				}
				if content != element {
					continue
				}
				if prio > 0 && prio != lprio {
					continue
				}
				delme = append(delme, id)
				continue
			}
		}
	}
	logging.Debug("Deleted ", len(delme), " rows from ", domain, "( ", name, "/", rtype, " ) by ", owner)
	_, e = d.transaction.Exec(`DELETE FROM records_api WHERE id = ANY($1)`, pq.Array(delme))
	n = len(delme)
	return
}

func (d *database) sqlInsertDomain(name string) (domainId int, e error) {
	domainId, e = d.sqlGetDomainId(name)
	if e == nil {
		return
	}
	_, e = d.transaction.Exec(`INSERT INTO domains(name,type) VALUES($1,$2)`,
		name, "NATIVE")
	if e != nil {
		return
	}
	domainId, e = d.sqlGetDomainId(name)
	return
}

func (d *database) sqlReplaceRecord(domain, name, rtype string, ttl int, data []string, owner string) (n int, e error) {
	if n, e = d.sqlDeleteRecord(domain, name, rtype, ttl, nil, owner); e == nil {
		if n == 0 {
			e = errors.New("you're trying to replace a record that doesn't exists, use create instead")
		} else if n, e = d.sqlInsertRecord(domain, name, rtype, ttl, data, owner); e == nil {
			e = nil
		}
	}
	return
}

func (d *database) sqlGetDomainRecords(domainName string, record []*Record, pretty bool) (records []interface{}, e error) {
	records = make([]interface{}, 0)
	var domainId int
	domainId, e = d.sqlGetDomainId(domainName)
	if e != nil {
		return
	}
	rows, e := d.transaction.Query(`SELECT name,type,content,ttl,prio FROM records WHERE domain_id=$1`, domainId)
	if e != nil {
		return
	}
	rmar := make(map[string]Record)
	rmap := make(map[string][]string)
	rhash := func(data ...interface{}) string {
		return fmt.Sprint(data) // TODO: make it more realistic
	}
	for rows.Next() {
		var name string
		var rtype string
		var content string
		var ttl int
		var prio int
		e = rows.Scan(&name, &rtype, &content, &ttl, &prio)
		if e != nil {
			continue
		}
		if record != nil {
			match := false
			for _, r := range record {
				if r != nil {
					if strings.HasPrefix(name, r.Name) {
						match = true
					}
				} else {
					match = true
				}
			}
			if !match {
				continue
			}
		}
		v := rhash(name, rtype, ttl)
		if _, ok := rmap[v]; !ok {
			rmap[v] = make([]string, 0)
		}

		if rtype == string(RecordTypeSRV) || rtype == string(RecordTypeMX) {
			content = fmt.Sprintf("%d %s", prio, content)
		}
		rmap[v] = append(rmap[v], content)
		if _, ok := rmar[v]; !ok {
			rmar[v] = Record{
				Type: RecordType(rtype),
				Name: name,
				TTL:  ttl,
				view: d.config.View,
			}
		}
	}
	for k, r := range rmar {
		if pretty {
			records = append(records, RecordPretty{
				Name: stripDomain(r.Name, domainName),
				Data: rmap[k],
				Type: r.Type,
				TTL:  r.TTL,
				view: r.view,
			})
		} else {
			records = append(records, Record{
				Name: stripDomain(r.Name, domainName),
				Data: rmap[k],
				Type: r.Type,
				TTL:  r.TTL,
				view: r.view,
			})
		}
	}
	return
}

func (d *database) sqlGetOwnRecords(ownerToken string, pretty bool) (records []interface{}, e error) {
	rows, e := d.transaction.Query(`SELECT name,type,content,ttl FROM records_api WHERE owner=$1`, ownerToken)
	if e != nil {
		return
	}
	rmar := make(map[string]Record)
	rmap := make(map[string][]string)
	rhash := func(data ...interface{}) string {
		return fmt.Sprint(data) // TODO: make it more realistic
	}
	for rows.Next() {
		var name string
		var rtype string
		var content string
		var ttl int
		e = rows.Scan(&name, &rtype, &content, &ttl)
		if e != nil {
			continue
		}
		v := rhash(name, rtype, ttl)
		if _, ok := rmap[v]; !ok {
			rmap[v] = make([]string, 0)
		}
		rmap[v] = append(rmap[v], content)
		if _, ok := rmar[v]; !ok {
			rmar[v] = Record{
				Type: RecordType(rtype),
				Name: name,
				TTL:  ttl,
				view: d.config.View,
			}
		}
	}
	for k, r := range rmar {
		if pretty {
			records = append(records, RecordPretty{
				Name: r.Name,
				Data: rmap[k],
				Type: r.Type,
				TTL:  r.TTL,
				view: r.view,
			})
		} else {
			records = append(records, Record{
				Name: r.Name,
				Data: rmap[k],
				Type: r.Type,
				TTL:  r.TTL,
				view: r.view,
			})
		}
	}
	return
}

func stripDomain(record, domain string) string {
	r2 := strings.TrimSuffix(record, domain)
	for strings.HasSuffix(r2, ".") {
		r2 = strings.TrimSuffix(r2, ".")
	}
	return r2
}
func (d *database) getDomains(pretty bool) (domains []interface{}, e error) {
	domains = make([]interface{}, 0)
	rows, e := d.transaction.Query(`SELECT name FROM domains`)
	if e != nil {
		return
	}
	for rows.Next() {
		var name string
		e = rows.Scan(&name)
		if e != nil {
			return
		}
		if pretty {
			domains = append(domains, DomainPretty{
				Name: name,
				View: d.config.View,
			})
		} else {
			domains = append(domains, Domain{
				Name: name,
				View: d.config.View,
			})
		}
	}
	return
}
