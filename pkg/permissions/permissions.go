/*
 * Copyright (c) 2022 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package permissions

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/configuration"
	"io"
	"log"
	"net/http"
	"net/url"
	"runtime/debug"
)

type Permissions struct {
	config configuration.Config
}

func New(config configuration.Config) *Permissions {
	return &Permissions{config: config}
}

func (this *Permissions) CheckAccess(token auth.Token, topic string, id string, right string) (allowed bool, err error) {
	req, err := http.NewRequest("GET", this.config.PermissionsUrl+"/v3/resources/"+url.QueryEscape(topic)+"/"+url.QueryEscape(id)+"/access?rights="+right, nil)
	if err != nil {
		debug.PrintStack()
		return false, err
	}
	req.Header.Set("Authorization", token.Jwt())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("ERROR: ", err)
		debug.PrintStack()
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		return false, errors.New(buf.String())
	}
	err = json.NewDecoder(resp.Body).Decode(&allowed)
	if err != nil {
		debug.PrintStack()
		return false, err
	}
	return allowed, nil
}

func (this *Permissions) Query(token string, query QueryMessage, result interface{}) (err error, code int) {
	requestBody := new(bytes.Buffer)
	err = json.NewEncoder(requestBody).Encode(query)
	if err != nil {
		return err, http.StatusInternalServerError
	}
	req, err := http.NewRequest("POST", this.config.PermissionsUrl+"/v3/query", requestBody)
	if err != nil {
		debug.PrintStack()
		return err, http.StatusInternalServerError
	}
	req.Header.Set("Authorization", token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		debug.PrintStack()
		return err, http.StatusInternalServerError
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		err = errors.New(buf.String())
		log.Println("ERROR: ", resp.StatusCode, err)
		debug.PrintStack()
		return err, resp.StatusCode
	}
	err = json.NewDecoder(resp.Body).Decode(result)
	if err != nil {
		debug.PrintStack()
		return err, http.StatusInternalServerError
	}

	return nil, http.StatusOK
}

func (this *Permissions) GetResourceRights(token string, topic string, id string) (rights ResourceRights, err error, code int) {
	req, err := http.NewRequest("GET", this.config.PermissionsUrl+"/v3/administrate/rights/"+url.PathEscape(topic)+"/"+url.PathEscape(id), nil)
	if err != nil {
		debug.PrintStack()
		return rights, err, http.StatusInternalServerError
	}
	req.Header.Set("Authorization", token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		debug.PrintStack()
		return rights, err, http.StatusInternalServerError
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		err = errors.New(buf.String())
		log.Println("ERROR: ", resp.StatusCode, err)
		debug.PrintStack()
		return rights, err, resp.StatusCode
	}
	err = json.NewDecoder(resp.Body).Decode(&rights)
	if err != nil {
		debug.PrintStack()
		return rights, err, http.StatusInternalServerError
	}

	return rights, nil, http.StatusOK
}

func (this *Permissions) SetResourceRights(token string, topic string, id string, rights ResourceRights, kafkaKey string) error {
	requestBody := new(bytes.Buffer)
	err := json.NewEncoder(requestBody).Encode(rights)
	if err != nil {
		return err
	}
	queryParameter := ""
	if kafkaKey != "" {
		queryParameter = "?key=" + url.QueryEscape(kafkaKey)
	}
	req, err := http.NewRequest("PUT", this.config.PermissionsCmdUrl+"/v3/administrate/rights/"+url.PathEscape(topic)+"/"+url.PathEscape(id)+queryParameter, requestBody)
	if err != nil {
		debug.PrintStack()
		return err
	}
	req.Header.Set("Authorization", token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		debug.PrintStack()
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		err = errors.New(buf.String())
		log.Println("ERROR: ", resp.StatusCode, err)
		debug.PrintStack()
		return err
	}
	io.ReadAll(resp.Body) //empty body to ensure reuse of connection
	return nil
}
