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
	"errors"
	"fmt"
	"github.com/asaskevich/govalidator"
	"net"
	"strconv"
	"strings"
)

var checkers = map[RecordType]func(*Record) error{
	RecordTypeA:     checkRecordTypeA,
	RecordTypeAAAA:  checkRecordTypeAAAA,
	RecordTypeCNAME: checkRecordTypeCNAME,
	RecordTypeTXT:   checkRecordTypeTXT,
	RecordTypeSRV:   checkRecordTypeSRV,
	RecordTypeMX:    checkRecordTypeMX,
	RecordTypeNS:    checkRecordTypeNS,
	RecordTypeSOA:   checkRecordTypeSOA,
	RecordTypePTR:   checkRecordTypePTR,
}

func (d *Domain) record2dns(r *Record) string {
	if r.Name == "." || r.Name == "@" || r.Name == "" {
		return d.Name
	}
	return fmt.Sprintf("%s.%s", r.Name, d.Name)
}
func checkRFCRequest(request *WunderRequest) error {
	for _, r := range request.Record {
		if r.TTL < 0 {
			return errors.New("ttl can't be lesser than 0")
		}
		if r.TTL == 0 {
			r.TTL = 600 // default
		}
		dns := request.Domain.record2dns(r)
		if strings.HasPrefix(dns, "*.") {
			parts1 := strings.Split(dns, "*.")
			if len(parts1) > 2  || parts1[0] != "" {
				return errors.New("Insufficient use of wildcard")
			}
			dns = parts1[1]
		}
		if !govalidator.IsDNSName(dns) {
			return errors.New(fmt.Sprintf("%s: not a valid DNS name", dns))
		}
		if f, ok := checkers[r.Type]; ok {
			if e := f(r); e != nil {
				return e
			}
		}
	}
	return nil
}

func checkRecordTypeA(r *Record) error {
	if len(r.Data) == 0 {
		return errors.New("A record must have at least one argument")
	}
	for _, d := range r.Data {
		if !govalidator.IsIPv4(d) {
			return errors.New(fmt.Sprintf("%s: not an ipv4", d))
		}
	}

	return nil
}

func checkRecordTypeAAAA(r *Record) error {
	if len(r.Data) == 0 {
		return errors.New("AAAA record must have at least one argument")
	}
	for _, d := range r.Data {
		if !govalidator.IsIPv6(d) {
			return errors.New(fmt.Sprintf("%s: not an ipv6", d))
		}
	}
	return nil
}

func checkRecordTypeCNAME(r *Record) error {
	if len(r.Data) != 1 {
		return errors.New("CNAME must have single value")
	}
	if r.Name == "" {
		return errors.New("CNAME can't be root domain record")
	}
	if strings.HasSuffix(r.Data[0], ".") {
		return errors.New("CNAME mustn't end with '.', it's always a full domain name only")
	}
	if !govalidator.IsDNSName(r.Data[0]) {
		return errors.New(fmt.Sprintf("%s is not a valid domain name", r.Data[0]))
	}
	return nil
}

func checkRecordTypeTXT(r *Record) error {
	if len(r.Data) == 0 {
		return errors.New("TXT record must have at least one argument")
	}
	for _, d := range r.Data {
		if len(d) > 255 {
			return errors.New("TXT records can't be > 255 characters length")
		}
		if !govalidator.IsASCII(d) {
			return errors.New("TXT records can't contain non-ascii characters")
		}
	}
	return nil
}

func checkRecordTypeSRV(r *Record) error {
	if len(r.Data) == 0 {
		return errors.New("SRV record must have at least one argument")
	}
	// _service._proto.name. TTL class SRV priority weight port target.
	parts := strings.Split(r.Name, ".")
	if len(parts) < 3 {
		return errors.New("SRV record name must match `_service._proto.name` pattern")
	}
	if !strings.HasPrefix(parts[0], "_") || !strings.HasPrefix(parts[1], "_") {
		return errors.New("SRV record name must match `_service._proto.name` pattern")
	}
	for _, d := range r.Data {
		parts = strings.Split(d, " ")
		if len(parts) != 4 {
			return errors.New("SRV record data must match `priority weight port target` pattern")
		}
		if _, e := strconv.Atoi(parts[0]); e != nil {
			return errors.New("SRV record data(weight) must be a number")
		}
		if _, e := strconv.Atoi(parts[1]); e != nil {
			return errors.New("SRV record data(priority) weight must be a number")
		}
		if p, e := strconv.Atoi(parts[2]); e != nil {
			return errors.New("SRV record data(priority) port must be a number")
		} else if p < 0 || p > 65535 {
			return errors.New("SRV record data(priority) port must be a number between 0 and 65535")
		}
		if !govalidator.IsDNSName(parts[3]) {
			return errors.New("SRV record data(target) must be a valid domain name")
		}
	}
	return nil
}

func checkRecordTypeMX(r *Record) error {
	if len(r.Data) == 0 {
		return errors.New("MX record must have at least one argument")
	}
	// example.com.		1936	IN	MX	10         blackmail.example.com
	for _, d := range r.Data {
		parts := strings.Split(d, " ")
		if len(parts) != 2 {
			return errors.New("MX record data must match `priority target` pattern")
		}
		if _, e := strconv.Atoi(parts[0]); e != nil {
			return errors.New("MX record data(priority) must be a number")
		}
		if !govalidator.IsDNSName(parts[1]) {
			return errors.New("MX record data(target) must be a valid domain name")
		}
	}
	return nil
}

func checkRecordTypeNS(r *Record) error {
	if len(r.Data) == 0 {
		return errors.New("NS record must have at least one argument")
	}
	for _, d := range r.Data {
		if !govalidator.IsDNSName(d) {
			return errors.New(fmt.Sprintf("%s is not a valid domain name", d))
		}
		if i, e := net.LookupIP(d); e != nil {
			return errors.New(fmt.Sprintf("can't lookup %s: %s", d, e.Error()))
		} else if len(i) < 1 {
			return errors.New(fmt.Sprintf("%s: records not found", d))
		}
	}
	return nil
}

func checkRecordTypeSOA(r *Record) error {
	if r.Name != "" {
		return errors.New("SOA record must have empty name")
	}
	// no, create records
	if len(r.Data) != 1 {
		return errors.New("SOA records must have single value")
	}
	soa := strings.Split(strings.TrimSpace(r.Data[0]), " ")
	//ns1.wargaming.net admins.wargaming.net 2020020403 900 600 86400 600
	if len(soa) != 7 {
		return errors.New("SOA record must have 7 fields: MNAME RNAME SERIAL REFRESH RETRY EXPIRE TTL")
	}
	if !govalidator.IsDNSName(soa[0]) {
		return errors.New("SOA record MNAME field is not a valid hostname")
	}
	if i, e := net.LookupIP(soa[0]); e != nil || len(i) == 0 {
		return errors.New("SOA record MNAME field can't be resolved")
	}
	if zoneSerial, e := strconv.Atoi(soa[2]); e == nil {
		if minSerial, e := generateNewSerial("0"); e == nil {
			if minSerialInt, e := strconv.Atoi(minSerial); e == nil {
				if zoneSerial < minSerialInt {
					return errors.New(fmt.Sprintf("SOA record SERIAL field minimum value is: %d", minSerialInt))
				}
			} else {
				return errors.New("unexpected error in date-based serial generation (2)")
			}
		} else {
			return errors.New("unexpected error in date-based serial generation (1)")
		}
	} else {
		return errors.New("SOA record SERIAL field must be in format: YYYYMMDDXX")
	}
	refreshInt, e := strconv.Atoi(soa[3])
	if e != nil {
		return errors.New("SOA record REFRESH field must be INT between 0 and 86400")
	}
	if refreshInt < 0 || refreshInt > 86400 {
		return errors.New("SOA record REFRESH field must be INT between 0 and 86400")
	}
	retryInt, e := strconv.Atoi(soa[4])
	if e != nil {
		return errors.New("SOA record RETRY field must be INT between 0 and 86400")
	}
	if retryInt < 0 || refreshInt > 86400 {
		return errors.New("SOA record RETRY field must be INT between 0 and 86400")
	}
	if retryInt >= refreshInt {
		return errors.New("SOA record RETRY field must be lesser than REFRESH")
	}

	expireInt, e := strconv.Atoi(soa[5])
	if e != nil {
		return errors.New("SOA record EXPIRE field must be INT between 0 and 172800")
	}
	if expireInt < 0 || expireInt > 172800 {
		return errors.New("SOA record EXPIRE field must be INT between 0 and 172800")
	}
	if expireInt <= (refreshInt + retryInt) {
		return errors.New("SOA record EXPIRE field must be greater than (REFRESH+RETRY)")
	}
	ttlInt, e := strconv.Atoi(soa[6])
	if e != nil {
		return errors.New("SOA record TTL field must be INT between 0 and 86400")
	}
	if ttlInt < 0 || ttlInt > 86400 {
		return errors.New("SOA record EXPIRE field must be INT between 0 and 172800")
	}
	return nil
}

func checkRecordTypePTR(r *Record) error {
	if len(r.Data) != 1 {
		return errors.New("only one PTR is allowed for one ip")
	}
	if !govalidator.IsDNSName(r.Data[0]) {
		return errors.New(fmt.Sprintf("%s is not a valid domain name", r.Data[0]))

	}
	return nil
}
