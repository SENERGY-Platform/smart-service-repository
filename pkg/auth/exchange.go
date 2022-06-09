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

package auth

import (
	"encoding/json"
	"errors"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/configuration"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"
)

func GetCachedTokenProvider(config configuration.Config) func(userid string) (token Token, err error) {
	cache := NewCache(config.TokenCacheSizeInMb*MB, config.TokenCacheDefaultExpirationInSeconds)
	return func(userid string) (token Token, err error) {
		err = cache.UseWithExpirationInResult("token."+userid, func() (interface{}, int, error) {
			return ExchangeUserToken(config, userid)
		}, &token)
		return
	}
}

func ExchangeUserToken(config configuration.Config, userid string) (token Token, expiration int, err error) {
	resp, err := http.PostForm(config.AuthEndpoint+"/auth/realms/master/protocol/openid-connect/token", url.Values{
		"client_id":         {config.AuthClientId},
		"client_secret":     {config.AuthClientSecret},
		"grant_type":        {"urn:ietf:params:oauth:grant-type:token-exchange"},
		"requested_subject": {userid},
	})
	if err != nil {
		return
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		log.Println("ERROR: GetUserToken()", resp.StatusCode, string(body))
		err = errors.New("access denied")
		resp.Body.Close()
		return
	}
	var openIdToken OpenidToken
	err = json.NewDecoder(resp.Body).Decode(&openIdToken)
	if err != nil {
		return
	}
	token, err = Parse("Bearer " + openIdToken.AccessToken)
	return token, int(openIdToken.ExpiresIn) - 5, err // subtract 5 seconds from expiration as a buffer
}

type OpenidToken struct {
	AccessToken      string    `json:"access_token"`
	ExpiresIn        float64   `json:"expires_in"`
	RefreshExpiresIn float64   `json:"refresh_expires_in"`
	RefreshToken     string    `json:"refresh_token"`
	TokenType        string    `json:"token_type"`
	RequestTime      time.Time `json:"-"`
}
