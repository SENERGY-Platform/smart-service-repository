/*
 * Copyright 2021 InfAI (CC SES)
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

package auth

import (
	"encoding/json"
	"errors"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/configuration"
	"log"
	"net/http"
	"time"

	"net/url"

	"io/ioutil"
)

func (this *OpenidToken) EnsureAccess(config configuration.Config) (token string, err error) {
	duration := time.Now().Sub(this.RequestTime).Seconds()

	if this.AccessToken != "" && this.ExpiresIn > duration+config.AuthExpirationTimeBuffer {
		token = "Bearer " + this.AccessToken
		return
	}

	if this.RefreshToken != "" && this.RefreshExpiresIn > duration+config.AuthExpirationTimeBuffer {
		log.Println("refresh token", this.RefreshExpiresIn, duration)
		openid, err := RefreshOpenidToken(config.AuthEndpoint, config.AuthClientId, config.AuthClientSecret, this.RefreshToken)
		if err != nil {
			log.Println("WARNING: unable to use refreshtoken", err)
		} else {
			*this = openid
			token = "Bearer " + this.AccessToken
			return token, err
		}
	}

	log.Println("get new access token")
	openid, err := GetOpenidToken(config.AuthEndpoint, config.AuthClientId, config.AuthClientSecret)
	*this = openid
	if err != nil {
		log.Println("ERROR: unable to get new access token", err)
		*this = OpenidToken{}
	}
	token = "Bearer " + this.AccessToken
	return
}

func GetOpenidToken(authEndpoint string, authClientId string, authClientSecret string) (openid OpenidToken, err error) {
	requesttime := time.Now()
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.PostForm(authEndpoint+"/auth/realms/master/protocol/openid-connect/token", url.Values{
		"client_id":     {authClientId},
		"client_secret": {authClientSecret},
		"grant_type":    {"client_credentials"},
	})
	if err != nil {
		return openid, err
	}
	if resp.StatusCode >= 300 {
		b, _ := ioutil.ReadAll(resp.Body)
		err = errors.New(resp.Status + ": " + string(b))
		return
	}
	err = json.NewDecoder(resp.Body).Decode(&openid)
	openid.RequestTime = requesttime
	return
}

func RefreshOpenidToken(authEndpoint string, authClientId string, authClientSecret string, refreshToken string) (openid OpenidToken, err error) {
	requesttime := time.Now()
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.PostForm(authEndpoint+"/auth/realms/master/protocol/openid-connect/token", url.Values{
		"client_id":     {authClientId},
		"client_secret": {authClientSecret},
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
	})

	if err != nil {
		return openid, err
	}
	if resp.StatusCode >= 300 {
		b, _ := ioutil.ReadAll(resp.Body)
		err = errors.New(resp.Status + ": " + string(b))
		return
	}
	err = json.NewDecoder(resp.Body).Decode(&openid)
	openid.RequestTime = requesttime
	return
}
