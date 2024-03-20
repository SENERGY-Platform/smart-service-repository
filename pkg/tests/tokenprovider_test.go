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

package tests

import (
	"encoding/json"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/configuration"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTokenProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		t.Log("DEBUG: auth call")
		json.NewEncoder(writer).Encode(auth.OpenidToken{
			AccessToken:      strings.TrimPrefix(userToken, "Bearer "),
			ExpiresIn:        time.Hour.Seconds(),
			RefreshExpiresIn: time.Hour.Seconds(),
			TokenType:        "",
		})
	}))
	defer server.Close()

	provider, err := auth.GetCachedTokenProvider(configuration.Config{AuthEndpoint: server.URL})
	if err != nil {
		t.Error(err)
		return
	}

	token, err := provider("user")
	if err != nil {
		t.Error(err)
		return
	}
	if token.Token == "" {
		t.Error("token is empty", token)
		return
	}

	token, err = provider("user")
	if err != nil {
		t.Error(err)
		return
	}
	if token.Token == "" {
		t.Error("token is empty")
		return
	}
}
