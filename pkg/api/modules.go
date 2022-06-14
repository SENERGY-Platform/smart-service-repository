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
	endpoints = append(endpoints, &Modules{})
}

type Modules struct{}

// List godoc
// @Summary      returns a list of smart-service models
// @Description  returns a list of smart-service models
// @Produce      json
// @Tags         modules
// @Param        module_type query string false "filter by module type"
// @Param        instance_id query string false "filter by instance id"
// @Param        limit query integer false "limits size of result; 0 means unlimited"
// @Param        offset query integer false "offset to be used in combination with limit"
// @Success      200 {array} model.SmartServiceModule
// @Failure      500
// @Failure      401
// @Router       /modules [get]
func (this *Modules) List(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.GET("/modules", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		query := model.ModuleQueryOptions{}
		moduleTypeFilter := request.URL.Query().Get("module_type")
		if moduleTypeFilter != "" {
			query.TypeFilter = &moduleTypeFilter
		}
		instanceIdFilter := request.URL.Query().Get("instance_id")
		if instanceIdFilter != "" {
			query.InstanceIdFilter = &instanceIdFilter
		}
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
			query.Sort = "id.asc"
		}

		result, err, code := ctrl.ListModules(token, query)
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(result)
	})
}

// CreateByProcessInstance godoc
// @Summary      create a smart-service module
// @Description  creates a smart-service module
// @Tags         modules
// @Accept       json
// @Produce      json
// @Param        id path string true "Process-Instance ID"
// @Param        message body model.SmartServiceModuleInit true "SmartServiceModuleInit"
// @Success      200 {object} model.SmartServiceModule
// @Failure      500
// @Failure      400
// @Failure      401
// @Router       /instances-by-process-id/{id}/modules [post]
func (this *Modules) CreateByProcessInstance(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.POST("/instances-by-process-id/:id/modules", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}
		if !token.IsAdmin() {
			http.Error(writer, "only admins may ask for instance user-id", http.StatusForbidden)
			return
		}

		module := model.SmartServiceModuleInit{}
		err = json.NewDecoder(request.Body).Decode(&module)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		result, err, code := ctrl.AddModuleForProcessInstance(params.ByName("id"), module)
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(result)
	})
}

// SetByProcessInstance godoc
// @Summary      set a smart-service module
// @Description  set a smart-service module; existing modules will be updated, missing modules will be created
// @Tags         modules
// @Accept       json
// @Produce      json
// @Param        id path string true "Process-Instance ID"
// @Param        moduleId path string true "Module ID"
// @Param        message body model.SmartServiceModuleInit true "SmartServiceModuleInit"
// @Success      200 {object} model.SmartServiceModule
// @Failure      500
// @Failure      400
// @Failure      401
// @Router       /instances-by-process-id/{id}/modules/{moduleId} [put]
func (this *Modules) SetByProcessInstance(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.PUT("/instances-by-process-id/:id/modules/:moduleId", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}
		if !token.IsAdmin() {
			http.Error(writer, "only admins may ask for instance user-id", http.StatusForbidden)
			return
		}

		module := model.SmartServiceModuleInit{}
		err = json.NewDecoder(request.Body).Decode(&module)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		result, err, code := ctrl.SetModuleForProcessInstance(params.ByName("id"), module, params.ByName("moduleId"))
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(result)
	})
}

// Create godoc
// @Summary      create a smart-service module
// @Description  creates a smart-service module
// @Tags         modules
// @Accept       json
// @Produce      json
// @Param        id path string true "Instance ID"
// @Param        message body model.SmartServiceModuleInit true "SmartServiceModuleInit"
// @Success      200 {object} model.SmartServiceModule
// @Failure      500
// @Failure      400
// @Failure      401
// @Router       /instances/{id}/modules [post]
func (this *Modules) Create(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.POST("/instances/:id/modules", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		module := model.SmartServiceModuleInit{}
		err = json.NewDecoder(request.Body).Decode(&module)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		result, err, code := ctrl.AddModule(token, params.ByName("id"), module)
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(result)
	})
}
