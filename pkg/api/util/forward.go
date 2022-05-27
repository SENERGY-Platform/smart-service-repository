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

package util

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

func NewConditionalForward(defaultHandler http.Handler, remote string, condition func(r *http.Request) bool) *ConditionalForward {
	return &ConditionalForward{defaultHandler: defaultHandler, condition: condition, remote: remote}
}

type ConditionalForward struct {
	defaultHandler http.Handler
	remote         string
	condition      func(r *http.Request) bool
}

func (this *ConditionalForward) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if this.condition(r) {
		this.forward(w, r)
	} else {
		this.defaultHandler.ServeHTTP(w, r)
	}
}

func (this *ConditionalForward) forward(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	url := this.remote
	if strings.HasPrefix(r.RequestURI, "/") {
		url = url + r.RequestURI
	} else {
		url = url + "/" + r.RequestURI
	}

	proxyReq, err := http.NewRequest(r.Method, url, bytes.NewReader(body))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	proxyReq.Header = r.Header

	resp, err := http.DefaultClient.Do(proxyReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
