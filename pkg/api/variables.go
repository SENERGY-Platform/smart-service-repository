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

package api

import (
	"encoding/json"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"strconv"
)

func init() {
	endpoints = append(endpoints, &Variables{})
}

type Variables struct{}

func (this *Variables) Set(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.PUT("/instances/:id/variables/:name", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		instanceId := params.ByName("id")
		if instanceId == "" {
			http.Error(writer, "missing instance id", http.StatusBadRequest)
			return
		}
		name := params.ByName("name")
		if name == "" {
			http.Error(writer, "missing variable name", http.StatusBadRequest)
			return
		}

		element := model.SmartServiceInstanceVariable{
			InstanceId: instanceId,
			UserId:     token.GetUserId(),
			Name:       name,
			Value:      nil,
		}
		err = json.NewDecoder(request.Body).Decode(&element.Value)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}

		result, err, code := ctrl.SetVariable(token, element)
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(result)
	})
}

func (this *Variables) Get(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.GET("/instances/:id/variables/:name", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		instanceId := params.ByName("id")
		if instanceId == "" {
			http.Error(writer, "missing instance id", http.StatusBadRequest)
			return
		}
		name := params.ByName("name")
		if name == "" {
			http.Error(writer, "missing variable name", http.StatusBadRequest)
			return
		}

		result, err, code := ctrl.GetVariable(token, instanceId, name)
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(result)
	})
}

func (this *Variables) GetValue(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.GET("/instances/:id/variables/:name/value", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		instanceId := params.ByName("id")
		if instanceId == "" {
			http.Error(writer, "missing instance id", http.StatusBadRequest)
			return
		}
		name := params.ByName("name")
		if name == "" {
			http.Error(writer, "missing variable name", http.StatusBadRequest)
			return
		}

		result, err, code := ctrl.GetVariable(token, instanceId, name)
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(result.Value)
	})
}

func (this *Variables) List(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.GET("/instances/:id/variables", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		instanceId := params.ByName("id")
		if instanceId == "" {
			http.Error(writer, "missing instance id", http.StatusBadRequest)
			return
		}

		query := model.VariableQueryOptions{}
		limit := request.URL.Query().Get("limit")
		if limit != "" {
			query.Limit, err = strconv.Atoi(limit)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusBadRequest)
				return
			}
		}
		offset := request.URL.Query().Get("offset")
		if offset != "" {
			query.Offset, err = strconv.Atoi(offset)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusBadRequest)
				return
			}
		}
		query.Sort = request.URL.Query().Get("sort")
		if query.Sort == "" {
			query.Sort = "name.asc"
		}

		result, err, code := ctrl.ListVariables(token, instanceId, query)
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(result)
	})
}

func (this *Variables) Map(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.GET("/instances/:id/variables-map", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		instanceId := params.ByName("id")
		if instanceId == "" {
			http.Error(writer, "missing instance id", http.StatusBadRequest)
			return
		}

		query := model.VariableQueryOptions{}
		limit := request.URL.Query().Get("limit")
		if limit != "" {
			query.Limit, err = strconv.Atoi(limit)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusBadRequest)
				return
			}
		}
		offset := request.URL.Query().Get("offset")
		if offset != "" {
			query.Offset, err = strconv.Atoi(offset)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusBadRequest)
				return
			}
		}
		query.Sort = request.URL.Query().Get("sort")
		if query.Sort == "" {
			query.Sort = "name.asc"
		}

		result, err, code := ctrl.GetVariablesMap(token, instanceId, query)
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(result)
	})
}

func (this *Variables) Delete(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.DELETE("/instances/:id/variables/:name", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}
		instanceId := params.ByName("id")
		if instanceId == "" {
			http.Error(writer, "missing instance id", http.StatusBadRequest)
			return
		}

		name := params.ByName("name")
		if name == "" {
			http.Error(writer, "missing variable name", http.StatusBadRequest)
			return
		}

		err, code := ctrl.DeleteVariable(token, instanceId, name)
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		writer.WriteHeader(http.StatusOK)
	})
}

// no existing variable is deleted
func (this *Instances) SetVariablesMapByProcessInstance(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.PUT("/instances-by-process-id/:id/variables-map", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}
		if !token.IsAdmin() {
			http.Error(writer, "only admins may ask for instance user-id", http.StatusForbidden)
			return
		}
		variables := map[string]interface{}{}
		err = json.NewDecoder(request.Body).Decode(&variables)
		if err != nil {
			http.Error(writer, "expect json encoded object in body", http.StatusBadRequest)
			return
		}
		err, code := ctrl.SetVariablesMapOfProcessInstance(params.ByName("id"), variables)
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		writer.WriteHeader(http.StatusOK)
	})
}

func (this *Instances) GetVariablesMapByProcessInstance(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.GET("/instances-by-process-id/:id/variables-map", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}
		if !token.IsAdmin() {
			http.Error(writer, "only admins may ask for instance user-id", http.StatusForbidden)
			return
		}
		result, err, code := ctrl.GetVariablesMapOfProcessInstance(params.ByName("id"))
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(result)
	})
}
