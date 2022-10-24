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
)

func init() {
	endpoints = append(endpoints, &Maintenance{})
}

type Maintenance struct{}

// ListMaintenanceProcedures godoc
// @Summary      lists smart-service maintenance procedure information
// @Description  lists smart-service maintenance procedure information
// @Tags         instances, maintenance-procedures
// @Param        id path string true "Instance ID"
// @Produce      json
// @Success      200 {array}  model.MaintenanceProcedure
// @Failure      500
// @Failure      401
// @Router       /instances/{id}/maintenance-procedures [get]
func (this *Maintenance) ListMaintenanceProcedures(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.GET("/instances/:id/maintenance-procedures", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}
		result, err, code := ctrl.GetMaintenanceProceduresOfInstance(token, params.ByName("id"))
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(result)
	})
}

// GetMaintenanceProcedure godoc
// @Summary      get smart-service maintenance procedure information
// @Description  get smart-service maintenance procedure information
// @Tags         instances, maintenance-procedures
// @Param        id path string true "Instance ID"
// @Param        public_event_id path string true "public event id of maintenance-procedure"
// @Produce      json
// @Success      200 {object}  model.MaintenanceProcedure
// @Failure      500
// @Failure      401
// @Router       /instances/{id}/maintenance-procedures/{public_event_id} [get]
func (this *Maintenance) GetMaintenanceProcedure(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.GET("/instances/:id/maintenance-procedures/:public_event_id", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}
		result, err, code := ctrl.GetMaintenanceProcedureOfInstance(token, params.ByName("id"), params.ByName("public_event_id"))
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(result)
	})
}

// GetMaintenanceProcedureParameters godoc
// @Summary      returns parameters of a smart-service maintenance procedure
// @Description  returns parameters of a smart-service maintenance procedure
// @Tags         instances, maintenance-procedures, parameter
// @Produce      json
// @Param        id path string true "Instance ID"
// @Param        public_event_id path string true "public event id of maintenance-procedure"
// @Success      200 {array} model.SmartServiceExtendedParameter
// @Failure      500
// @Failure      401
// @Router       /instances/{id}/maintenance-procedures/{public_event_id}/parameters [get]
func (this *Maintenance) GetMaintenanceProcedureParameters(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.GET("/instances/:id/maintenance-procedures/:public_event_id/parameters", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}
		result, err, code := ctrl.GetMaintenanceProcedureParametersOfInstance(token, params.ByName("id"), params.ByName("public_event_id"))
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(result)
	})
}

// Start godoc
// @Summary      start a smart-service instance maintenance procedure
// @Description  start a smart-service instance maintenance procedure
// @Tags         instances, maintenance-procedures
// @Accept       json
// @Produce      json
// @Param        id path string true "Instance ID"
// @Param        public_event_id path string true "public event id of maintenance-procedure"
// @Param        message body model.SmartServiceParameters true "SmartServiceParameters"
// @Success      204
// @Failure      500
// @Failure      401
// @Router       /instances/{id}/maintenance-procedures/{public_event_id}/start [post]
func (this *Maintenance) Start(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.POST("/instances/:id/maintenance-procedures/:public_event_id/start", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}
		id := params.ByName("id")
		if id == "" {
			http.Error(writer, "missing release id", http.StatusBadRequest)
			return
		}
		publicEventId := params.ByName("public_event_id")
		if id == "" {
			http.Error(writer, "missing release id", http.StatusBadRequest)
			return
		}
		parameters := model.SmartServiceParameters{}
		err = json.NewDecoder(request.Body).Decode(&parameters)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		err, code := ctrl.StartMaintenanceProcedure(token, id, publicEventId, parameters)
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		writer.WriteHeader(http.StatusNoContent)
	})
}
