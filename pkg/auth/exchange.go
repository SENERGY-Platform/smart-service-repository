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
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/SENERGY-Platform/service-commons/pkg/cache"
	"github.com/SENERGY-Platform/service-commons/pkg/cache/localcache"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/configuration"
)

func GetCachedTokenProvider(config configuration.Config) (func(userid string) (token Token, err error), error) {
	c, err := cache.New(cache.Config{
		L1Provider: localcache.NewProvider(time.Duration(config.TokenCacheDefaultExpirationInSeconds)*time.Second, time.Second),
	})
	if err != nil {
		return nil, err
	}
	return func(userid string) (token Token, err error) {
		return cache.UseWithExpInGet(c, "token."+userid, func() (Token, time.Duration, error) {
			temp, exp, err := ExchangeUserToken(config, userid)
			return temp, time.Duration(exp) * time.Second, err
		}, func(token Token) error {
			if token.Jwt() == "" {
				return errors.New("invalid token loaded from cache")
			}
			return nil
		}, time.Duration(config.TokenCacheDefaultExpirationInSeconds)*time.Second)
	}, nil
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
		body, _ := io.ReadAll(resp.Body)
		config.GetLogger().Error("error in GetUserToken()", "error", resp.Status+": "+string(body))
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
