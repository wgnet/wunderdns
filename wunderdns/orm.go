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
package wunderdns

import (
	"errors"
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"strconv"
	"strings"
)

type domainTable struct {
	Id             uint           `gorm:"primaryKey"`
	Name           string         `gorm:"size:255;not null"`
	Master         *string        `gorm:"size:128"`
	LastCheck      *int           `gorm:"column:last_check"`
	Type           string         `gorm:"size:6;not null"`
	NotifiedSerial *string        `gorm:"column:notified_serial"`
	Account        *string        `gorm:"size:40"`
	Records        []RecordsTable `gorm:"foreignKey:DomainId"`
}

type RecordsTable struct {
	Id         uint   `gorm:"primaryKey"`
	DomainId   uint   `gorm:"column:domain_id"`
	Name       string `gorm:"size:255;not null"`
	Type       string `gorm:"size:10;not null"`
	Content    string `gorm:"size:65535;not null"`
	Ttl        *int
	Prio       *int
	changeDate *int `gorm:"column:change_date"`
	Disabled   *bool
	Ordername  *string `gorm:"size:255"`
	Auth       *bool
}

type RecordsApiTable struct {
	Id         uint   `gorm:"primaryKey"`
	DomainId   uint   `gorm:"column:domain_id"`
	Name       string `gorm:"size:255;not null"`
	Type       string `gorm:"size:10;not null"`
	Content    string `gorm:"size:65535;not null"`
	Ttl        *int
	Prio       *int
	changeDate *int `gorm:"column:change_date"`
	Disabled   *bool
	Ordername  *string `gorm:"size:255"`
	Auth       *bool
	Owner      *string `gorm:"size:255"`
}

func (domainTable) TableName() string {
	return "domains"
}

func (RecordsTable) TableName() string {
	return "records"
}

func (RecordsApiTable) TableName() string {
	return "records_api"
}

type orm struct {
	config    *PSQLConfig
	db        *gorm.DB
	lastError error
}

func ormApplyCommandData(tx *gorm.DB, view DomainView, request *WunderRequest) (data []interface{}, e error) {
	data = make([]interface{}, 0)
	switch request.Cmd {
	case CommandListDomains:
		var domains []domainTable
		tx.Find(&domains)
		for _, d := range domains {
			if request.Pretty {
				data = append(data, DomainPretty{
					Name: d.Name,
					View: view,
				})
			} else {
				data = append(data, Domain{
					Name: d.Name,
					View: view,
				})
			}
		}

		return
	case CommandSearchRecord:
		var records []RecordsTable
		var eq string
		// get record name from request
		if request.Record != nil && len(request.Record) > 0 {
			eq = request.Record[0].Name
		}
		tx.Where("name = ?", eq).Find(&records)
		recordsMap := make(map[string]*Record)

		for _, r := range records {
			hash := fmt.Sprintf("%s@%s", r.Type, r.Name)
			ttl := 0
			if r.Ttl != nil {
				ttl = *r.Ttl
			}
			if _, ok := recordsMap[hash]; ok {
				recordsMap[hash].Data = append(recordsMap[hash].Data, r.Content)
			} else {
				recordsMap[hash] = &Record{
					Name: r.Name,
					Type: RecordType(r.Type),
					Data: []string{r.Content},
					TTL:  ttl,
					view: view,
				}
			}
		}
		for _, r := range recordsMap {
			if request.Pretty {
				data = append(data, RecordPretty{
					Name: r.Name,
					Type: r.Type,
					Data: r.Data,
					TTL:  r.TTL,
					view: r.view,
				})
			} else {
				data = append(data, Record{
					Name: r.Name,
					Type: r.Type,
					Data: r.Data,
					TTL:  r.TTL,
					view: r.view,
				})
			}
		}
		return
	case CommandListRecords, CommandListOwn:
		var records []RecordsApiTable
		if request.Cmd == CommandListOwn {
			tx.Where("owner = ?", request.Auth.Token).Find(&records)
		} else {
			var r []RecordsTable
			var d domainTable
			tx.Where("name = ?", request.Domain.Name).First(&d)
			tx.Where("domain_id = ?", d.Id).Find(&r)
			records = make([]RecordsApiTable, len(r))
			for i, _r := range r {
				records[i] = RecordsApiTable{
					Id:         _r.Id,
					DomainId:   _r.DomainId,
					Name:       _r.Name,
					Type:       _r.Type,
					Content:    _r.Content,
					Ttl:        _r.Ttl,
					Prio:       _r.Prio,
					changeDate: _r.changeDate,
					Disabled:   _r.Disabled,
					Ordername:  _r.Ordername,
					Auth:       _r.Auth,
					Owner:      &request.Auth.Token,
				}
			}
		}
		recordsMap := make(map[string]*Record)
		for _, r := range records {
			// match records if any
			match := true
			if request.Record != nil && len(request.Record) > 0 {
				match = false
				for _, rr := range request.Record {
					if strings.HasPrefix(r.Name, rr.Name) {
						match = true
						break
					}
				}
			}
			if !match {
				continue
			}

			ttl := 0
			if r.Ttl != nil {
				ttl = *r.Ttl
			}
			prio := 0
			if r.Prio != nil {
				prio = *r.Prio
			}
			if r.Type == string(RecordTypeSRV) || r.Type == string(RecordTypeMX) {
				r.Content = fmt.Sprintf("%d %s", prio, r.Content)
			}
			hash := fmt.Sprintf("%s@%s", r.Type, r.Name)

			if _, ok := recordsMap[hash]; ok {
				recordsMap[hash].Data = append(recordsMap[hash].Data, r.Content)
			} else {
				recordsMap[hash] = &Record{
					Name: r.Name,
					Type: RecordType(r.Type),
					Data: []string{r.Content},
					TTL:  ttl,
					view: view,
				}
			}
		}

		for _, r := range recordsMap {

			if request.Pretty {
				data = append(data, RecordPretty{
					Name: r.Name,
					Type: r.Type,
					Data: r.Data,
					TTL:  r.TTL,
					view: r.view,
				})
			} else {
				data = append(data, Record{
					Name: r.Name,
					Type: r.Type,
					Data: r.Data,
					TTL:  r.TTL,
					view: r.view,
				})
			}
		}
		return
	default:
		e = errors.New("not implemented yet")
		return
	}
}

func ormApplyCommandExec(tx *gorm.DB, view DomainView, request *WunderRequest) (n int, e error) {

	switch request.Cmd {
	case CommandReplaceOwner:
		authDataLock.RLock()
		defer authDataLock.RUnlock()
		if e := globalConfig.Auth.checkIfCanMigrate(request); e != nil {
			return 0, e
		}
		var d domainTable
		tx.Where("name = ?", request.Domain.Name).First(&d)
		if d.Id == 0 {
			return 0, errors.New("domain not found")
		}
		for _, r := range request.Record {
			recordName := r.Name
			if r.Name == "." || r.Name == "@" || r.Name == "" {
				recordName = d.Name
			} else {
				recordName = fmt.Sprintf("%s.%s", r.Name, d.Name)
			}
			var recs []RecordsApiTable
			tx.Where("domain_id = ? and type = ? and name = ? and owner = ?", d.Id, r.Type,
				recordName, request.Auth.Token).Find(&recs)
			if len(recs) == 0 {
				return 0, errors.New("replace_owner: record not found")
			}
			for _, rec := range recs {
				rec.Owner = &request.NewToken
				n += int(tx.Updates(&rec).RowsAffected)
			}
		}
	case CommandCreateDomain:
		var d domainTable
		n = int(tx.FirstOrCreate(&d, domainTable{Name: request.Domain.Name, Type: "NATIVE"}).RowsAffected)
		return
	case CommandCreateRecord:
		var d domainTable
		tx.Where("name = ?", request.Domain.Name).First(&d)
		if d.Id == 0 {
			return 0, errors.New("domain not found")
		}
		for _, r := range request.Record {
			recordName := r.Name

			if r.Name == "." || r.Name == "@" || r.Name == "" {
				recordName = d.Name
			} else {
				recordName = fmt.Sprintf("%s.%s", r.Name, d.Name)
			}
			if r.Data == nil || len(r.Data) == 0 {
				return 0, errors.New("create_record: data is empty")
			}

			var exists []RecordsTable
			tx.Where("domain_id = ? and name = ?", d.Id, recordName).Find(&exists)
			for _, er := range exists {
				if strings.ToUpper(er.Type) == string(RecordTypeCNAME) {
					e = errors.New("(sql) multiple CNAME declaration")
					return
				}
				if r.Type == RecordTypeCNAME {
					e = errors.New("(json) multiple CNAME declaration")
					return
				}
				if strings.ToUpper(er.Type) == string(r.Type) && r.Type == RecordTypePTR {
					e = errors.New("(sql) multiple PTR declaration")
					return
				}
				if strings.ToUpper(er.Type) == string(r.Type) && r.Type == RecordTypeSOA {
					e = errors.New("(sql) multiple SOA declaration")
					return
				}
			}
			prio := 0
			for i := range r.Data {
				Content := r.Data[i]
				if r.Type == RecordTypeMX {
					mx := strings.Split(r.Data[i], " ")
					prio, _ = strconv.Atoi(mx[0])
					Content = mx[1]
				} else if r.Type == RecordTypeSRV {
					srv := strings.Split(r.Data[i], " ")
					prio, _ = strconv.Atoi(srv[0])
					Content = strings.Join(srv[1:], " ")
				}
				// create record
				logging.Info("Creating record ", recordName, r.Type, Content)
				_disabled := false
				_auth := true
				n += int(tx.Create(&RecordsApiTable{
					DomainId: d.Id,
					Name:     recordName,
					Type:     string(r.Type),
					Content:  Content,
					Ttl:      &r.TTL,
					Prio:     &prio,
					Disabled: &_disabled,
					Auth:     &_auth,
					Owner:    &request.Auth.Token,
				}).RowsAffected)
			}
		}
		if n > 0 {
			e = ormUpdateSOA(tx, d.Id)
		}
		return
	case CommandDeleteRecord:
		var d domainTable
		tx.Where("name = ?", request.Domain.Name).First(&d)
		if d.Id == 0 {
			return 0, errors.New("domain not found")
		}
		for _, r := range request.Record {
			recordName := r.Name

			if r.Name == "." || r.Name == "@" || r.Name == "" {
				recordName = d.Name
			} else {
				recordName = fmt.Sprintf("%s.%s", r.Name, d.Name)
			}
			if r.Data == nil || len(r.Data) == 0 {
				n += int(tx.Where("domain_id = ? and type = ? and name = ? and owner = ?", d.Id, r.Type,
					recordName, request.Auth.Token).Delete(&RecordsApiTable{}).RowsAffected)
			} else {
				for i := range r.Data {
					prio := 0
					if r.Type == RecordTypeMX {
						mx := strings.Split(r.Data[i], " ")
						prio, _ = strconv.Atoi(mx[0])
						r.Data[i] = mx[1]
					} else if r.Type == RecordTypeSRV {
						srv := strings.Split(r.Data[i], " ")
						prio, _ = strconv.Atoi(srv[0])
						r.Data[i] = strings.Join(srv[1:], " ")
					}
					dr := RecordsApiTable{
						DomainId: d.Id,
						Name:     recordName,
						Type:     string(r.Type),
						Content:  r.Data[i],
						Owner:    &request.Auth.Token,
					}
					if r.TTL != 0 {
						dr.Ttl = &r.TTL
					}
					if prio != 0 {
						dr.Prio = &prio
					}
					n += int(
						tx.Delete(&RecordsApiTable{}, &dr).RowsAffected,
					)

				}
			}
		}
		if n > 0 {
			e = ormUpdateSOA(tx, d.Id)
		}
		return
		//d.sqlUpdateSOA(request.Domain.Name)
	case CommandReplaceRecord:
		var d domainTable
		tx.Where("name = ?", request.Domain.Name).First(&d)
		if d.Id == 0 {
			return 0, errors.New("domain not found")
		}
		for _, r := range request.Record {
			recordName := r.Name

			if r.Name == "." || r.Name == "@" || r.Name == "" {
				recordName = d.Name
			} else {
				recordName = fmt.Sprintf("%s.%s", r.Name, d.Name)
			}
			if r.Data == nil || len(r.Data) == 0 {
				return 0, errors.New("replace_record: data is empty")
			}
			_n := int(tx.Where("domain_id = ? and type = ? and name = ? and owner = ?", d.Id, r.Type,
				recordName, request.Auth.Token).Delete(&RecordsApiTable{}).RowsAffected)
			if _n == 0 {
				return 0, errors.New("replace_record: no such record; create new record instead")
			} else {
				n += _n
				prio := 0
				for i := range r.Data {
					Content := r.Data[i]
					if r.Type == RecordTypeMX {
						mx := strings.Split(r.Data[i], " ")
						prio, _ = strconv.Atoi(mx[0])
						Content = mx[1]
					} else if r.Type == RecordTypeSRV {
						srv := strings.Split(r.Data[i], " ")
						prio, _ = strconv.Atoi(srv[0])
						Content = strings.Join(srv[1:], " ")
					}
					// create record
					logging.Info("Creating record ", r.view, recordName, r.Type, Content)
					_disabled := false
					_auth := true
					n += int(tx.Create(&RecordsApiTable{
						DomainId: d.Id,
						Name:     recordName,
						Type:     string(r.Type),
						Content:  Content,
						Ttl:      &r.TTL,
						Prio:     &prio,
						Disabled: &_disabled,
						Auth:     &_auth,
						Owner:    &request.Auth.Token,
					}).RowsAffected)
				}

			}
		}
		if n > 0 {
			e = ormUpdateSOA(tx, d.Id)
		}

	default:
		e = errors.New("not implemented yet")

	}
	return
}

func ormUpdateSOA(tx *gorm.DB, domainId uint) error {
	var r RecordsTable
	var e error
	tx.Where("domain_id = ? and type = ?", domainId, "SOA").First(&r)
	if r.Id == 0 {
		return errors.New("SOA record not found - create SOA record first")
	}
	r.Content = strings.TrimSpace(r.Content)
	parts := strings.Split(r.Content, " ")
	parts[2], e = generateNewSerial(parts[2]) // update serial
	if e != nil {
		return e
	}
	r.Content = strings.Join(parts, " ")
	tx.Save(&r)
	logging.Info("Updating soa for ", r.Name, " to ", parts[2])
	return nil
}

var orms = make([]*orm, 0)

func initORMs() error {
	for _, c := range globalConfig.PSQLConfigs {
		db, e := gorm.Open(postgres.Open(c.connString()), &gorm.Config{
			SkipDefaultTransaction: true,
		})
		if e != nil {
			return e
		}
		orms = append(orms, &orm{
			config: c,
			db:     db,
		})
		logging.Info("Found database: ", c.Host)
	}
	return nil
}

func ormApplyCommandMerge(request *WunderRequest) (re map[DomainView][]interface{}, e error) {
	logging.Info("[ormApplyCommandMerge]", request.toString())
	re = make(map[DomainView][]interface{})
	for _, d := range orms {
		if d.config.View != request.Domain.View && request.Domain.View != DomainViewAny {
			continue // skip
		}
		if _, ok := re[d.config.View]; !ok {
			re[d.config.View] = make([]interface{}, 0)
		}
		e = d.db.Transaction(func(tx *gorm.DB) error {
			data, e := ormApplyCommandData(tx, d.config.View, request)
			if e == nil {
				re[d.config.View] = append(re[d.config.View], data...)
			}
			return e
		})
		if e != nil {
			return
		}
	}
	return
}

func ormApplyCommand(request *WunderRequest) (n int, e error) {
	logging.Info("[ormApplyCommand]", request.toString())
	n = 0
	for _, d := range orms {
		if d.config.View != request.Domain.View && request.Domain.View != DomainViewAny {
			continue // skip
		}
		e = d.db.Transaction(func(tx *gorm.DB) error {
			_n, e := ormApplyCommandExec(tx, d.config.View, request)
			if e == nil {
				n += _n
			}
			return e
		})
		if e != nil {
			return
		}
	}
	return
}
