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
	"log"
	"net/http"
	"strconv"
)

func init() {
	endpoints = append(endpoints, &Instances{})
}

type Instances struct{}

// List godoc
// @Summary      returns a list of smart-service instances
// @Description  returns a list of smart-service instances
// @Tags         instances
// @Param        limit query integer false "limits size of result; 0 means unlimited"
// @Param        offset query integer false "offset to be used in combination with limit"
// @Param        sort query string false "describes the sorting in the form of name.asc"
// @Produce      json
// @Success      200 {array}  model.SmartServiceInstance
// @Failure      500
// @Failure      401
// @Router       /instances [get]
func (this *Instances) List(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.GET("/instances", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}
		query := model.InstanceQueryOptions{}
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
		result, err, code := ctrl.ListInstances(token, query)
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(result)
	})
}

// Get godoc
// @Summary      returns a smart-service instance
// @Description  returns a smart-service instance
// @Tags         instances
// @Produce      json
// @Param        id path string true "Instance ID"
// @Success      200 {object}  model.SmartServiceInstance
// @Failure      500
// @Failure      401
// @Router       /instances/{id} [get]
func (this *Instances) Get(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.GET("/instances/:id", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}
		id := params.ByName("id")
		if id == "" {
			http.Error(writer, "missing id", http.StatusBadRequest)
			return
		}
		result, err, code := ctrl.GetInstance(token, id)
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(result)
	})
}

// Delete godoc
// @Summary      removes a smart-service instance with all modules
// @Description  removes a smart-service instance with all modules
// @Tags         instances
// @Param        id path string true "Instance ID"
// @Success      200
// @Failure      500
// @Failure      401
// @Router       /instances/{id} [delete]
func (this *Instances) Delete(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.DELETE("/instances/:id", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}
		id := params.ByName("id")
		if id == "" {
			http.Error(writer, "missing id", http.StatusBadRequest)
			return
		}

		ignoreModuleDeleteErrors, _ := strconv.ParseBool(request.URL.Query().Get("ignore_module_delete_errors"))

		err, code := ctrl.DeleteInstance(token, id, ignoreModuleDeleteErrors)
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		writer.WriteHeader(http.StatusOK)
	})
}

// SetError godoc
// @Summary      sets smart-service instance error
// @Description  sets smart-service instance error
// @Tags         instances, error
// @Accept       json
// @Param        id path string true "Instance ID"
// @Param        message body string true "error message (json encoded)"
// @Success      200
// @Failure      500
// @Failure      401
// @Router       /instances/{id}/error [put]
func (this *Instances) SetError(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.PUT("/instances/:id/error", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}
		errMsg := ""
		err = json.NewDecoder(request.Body).Decode(&errMsg)
		if err != nil {
			http.Error(writer, "expect json encoded string in body", http.StatusBadRequest)
			return
		}
		err, code := ctrl.SetInstanceError(token, params.ByName("id"), errMsg)
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		writer.WriteHeader(http.StatusOK)
	})
}

// SetErrorByProcessInstance godoc
// @Summary      sets smart-service instance error
// @Description  sets smart-service instance error
// @Tags         instances, process-id, error
// @Accept       json
// @Param        id path string true "Process-Instance ID"
// @Param        message body string true "error message (json encoded)"
// @Success      200
// @Failure      500
// @Failure      401
// @Router       /instances-by-process-id/{id}/error [put]
func (this *Instances) SetErrorByProcessInstance(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.PUT("/instances-by-process-id/:id/error", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}
		errMsg := ""
		err = json.NewDecoder(request.Body).Decode(&errMsg)
		if err != nil {
			http.Error(writer, "expect json encoded string in body", http.StatusBadRequest)
			return
		}
		err, code := ctrl.SetInstanceErrorByProcessInstanceId(token, params.ByName("id"), errMsg)
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		writer.WriteHeader(http.StatusOK)
	})
}

// Redeploy godoc
// @Summary      updates smart-service instance parameter
// @Description  updates smart-service instance parameter
// @Tags         instances, parameter
// @Accept       json
// @Produce      json
// @Param        id path string true "Instance ID"
// @Param        message body model.SmartServiceParameters true "SmartServiceParameter"
// @Success      200 {object}  model.SmartServiceInstance
// @Failure      500
// @Failure      401
// @Router       /instances/{id}/parameters [put]
func (this *Instances) Redeploy(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.PUT("/instances/:id/parameters", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		//TODO: replace with real code
		log.Println(token)
		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(model.SmartServiceInstance{})
	})
}

// UpdateInfo godoc
// @Summary      updates smart-service instance parameter
// @Description  updates smart-service instance parameter
// @Tags         instances, parameter
// @Accept       json
// @Produce      json
// @Param        id path string true "Instance ID"
// @Param        message body model.SmartServiceInstanceInfo true "SmartServiceParameter"
// @Success      200 {object}  model.SmartServiceInstance
// @Failure      500
// @Failure      401
// @Router       /instances/{id}/info [put]
func (this *Instances) UpdateInfo(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.PUT("/instances/:id/info", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		//TODO: replace with real code
		log.Println(token)
		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(model.SmartServiceInstance{})
	})
}
