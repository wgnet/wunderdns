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
	"crypto"
	"errors"
	"fmt"
	"strings"
	"time"
)

func securityProcessRequest(request *WunderRequest) error {
	authDataLock.RLock()
	defer authDataLock.RUnlock()
	if request == nil || request.Auth == nil || request.Domain == nil {
		return errors.New("[auth] invalid (null) request")
	}
	if !globalConfig.Auth.checkAuthentication(request) {
		return errors.New(fmt.Sprintf("[auth] %s - invalid token/secret", request.Auth.Token))
	}
	if !globalConfig.Auth.isPermitted(request) {
		return errors.New(fmt.Sprintf("[auth] %s @ %s -> %s/%s - permission denied", request.Auth.Token, request.Cmd, request.Domain.Name,
			request.Domain.View))
	}
	return nil
}

func checkDomainMatch(one, other *Domain) bool {
	viewOk := false
	nameOk := false
	if one.View == other.View || one.View == DomainViewAny {
		viewOk = true
	}
	if one.Name == other.Name || one.Name == DomainNameAny {
		nameOk = true
	}
	if strings.HasPrefix(one.Name, "*") && strings.HasSuffix(other.Name, one.Name[1:]) {
		nameOk = true
	}

	return viewOk && nameOk
}
func (authDatabase *AuthDatabase) isPermitted(request *WunderRequest) bool {
	v, ok := (*authDatabase)[request.Auth.Token]
	if !ok {
		return false
	}
	if request.Cmd == CommandListOwn { // commit changes
		request.Auth.priority = v.Priority
		return true
	}
	for _, p := range v.Permissions {
		if checkDomainMatch(&p.Domain, request.Domain) {
			for _, c := range p.Permitted {
				if c == request.Cmd || c == CommandAny {
					request.Auth.priority = v.Priority // commit changes
					return true
				}
			}
		}
	}
	return false
}

/**
 * CRYPTO SHIT HERE
 * NEVER ROLL YOUR OWN CRYPTO
 * NEVER EVER
 */
func (authDatabase *AuthDatabase) checkAuthentication(request *WunderRequest) bool {
	if request.Auth == nil {
		logging.Debug("[auth] auth header is null")
		return false
	}
	if request.Auth.Token == "" || request.Auth.Sum == "" {
		logging.Debug("[auth] auth token is null")
		return false
	}
	if v, ok := (*authDatabase)[request.Auth.Token]; !ok {
		logging.Debug("[auth] token not found in database: ", request.Auth.Token)
		return false
	} else {
		for shift := -900; shift <= 900; shift += 30 {
			h := createVariodicHash(request, shift)
			x := crypto.SHA256.New()
			x.Write([]byte(fmt.Sprintf("%s@%s", v.Secret, h)))
			x2 := fmt.Sprintf("%0x", x.Sum(nil))
			if x2 == request.Auth.Sum {
				return true
			}
		}
	}
	logging.Debug("[auth] invalid hash for token: ", request.Auth.Token)
	return false
}

func (authDatabase *AuthDatabase) checkIfCanMigrate(request *WunderRequest) error {
	if request.NewToken != "" {
		if _, ok := (*authDatabase)[request.NewToken]; !ok {
			return errors.New("new token doesn't exist in our database")
		}
	} else {
		return errors.New("new token is empty")
	}
	return nil
}

func createVariodicHash(request *WunderRequest, shift int) string {
	t := time.Now().Unix()
	t -= t % 30
	t += int64(shift)
	s := new(strings.Builder)
	if request.Domain != nil {
		s.WriteString(string(request.Domain.View))
		s.WriteString(request.Domain.Name)
	}
	s.WriteString(string(request.Cmd))
	if request.Record != nil {
		for _, n := range request.Record {
			s.WriteString(string(n.Type))
			s.WriteString(n.Name)
			s.WriteString(strings.Join(n.Data, "@"))
		}
	}
	s.WriteString(fmt.Sprintf("%d", t))
	x := crypto.SHA256.New()
	x.Write([]byte(s.String()))
	return fmt.Sprintf("%0x", x.Sum(nil))
}
