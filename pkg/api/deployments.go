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
)

func init() {
	endpoints = append(endpoints, &Deployments{})
}

type Deployments struct{}

// List godoc
// @Summary      returns a list of smart-service deployments
// @Description  returns a list of smart-service deployments
// @Tags         deployments
// @Produce      json
// @Success      200 {array} model.SmartServiceDeployment
// @Failure      500
// @Failure      401
// @Router       /deployments [get]
func (this *Deployments) List(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.GET("/deployments", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		//TODO: replace with real code
		log.Println(token)
		json.NewEncoder(writer).Encode([]model.SmartServiceDeployment{})
	})
}

// Get godoc
// @Summary      returns a smart-service deployment
// @Description  returns a smart-service deployment
// @Tags         deployments
// @Produce      json
// @Param        id path string true "Deployment ID"
// @Success      200 {object} model.SmartServiceDeployment
// @Failure      500
// @Failure      401
// @Router       /deployments/{id} [get]
func (this *Deployments) Get(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.GET("/deployments/:id", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		//TODO: replace with real code
		log.Println(token)
		json.NewEncoder(writer).Encode(model.SmartServiceDeployment{})
	})
}

// Update godoc
// @Summary      updates a smart-service deployment
// @Description  updates a smart-service deployment
// @Tags         deployments
// @Accept       json
// @Produce      json
// @Param        id path string true "Deployment ID"
// @Param        message body model.SmartServiceDeployment true "SmartServiceDeployment"
// @Success      200 {object} model.SmartServiceDeployment
// @Failure      500
// @Failure      401
// @Router       /deployments/{id} [put]
func (this *Deployments) Update(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.PUT("/deployments/:id", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		//TODO: replace with real code
		log.Println(token)
		json.NewEncoder(writer).Encode(model.SmartServiceDeployment{})
	})
}

// Create godoc
// @Summary      create a smart-service deployment
// @Description  creates a smart-service deployment
// @Tags         deployments
// @Accept       json
// @Produce      json
// @Param        id path string true "Deployment ID"
// @Param        message body model.SmartServiceDeployment true "SmartServiceDeployment"
// @Success      200 {object} model.SmartServiceDeployment
// @Failure      500
// @Failure      401
// @Router       /deployments [post]
func (this *Deployments) Create(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.POST("/deployments", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		//TODO: replace with real code
		log.Println(token)
		json.NewEncoder(writer).Encode(model.SmartServiceDeployment{})
	})
}

// Delete godoc
// @Summary      removes a smart-service deployment
// @Description  removes a smart-service deployment
// @Tags         deployments
// @Accept       json
// @Produce      json
// @Param        id path string true "Deployment ID"
// @Success      200
// @Failure      500
// @Failure      401
// @Router       /deployments/{id} [delete]
func (this *Deployments) Delete(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.POST("/deployments/:id", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		//TODO: replace with real code
		log.Println(token)
		json.NewEncoder(writer).Encode(model.SmartServiceDeployment{})
	})
}

// Parameters godoc
// @Summary      returns parameters of a deployment
// @Description  returns parameters of a deployment
// @Tags         deployments, parameter
// @Produce      json
// @Param        id path string true "Deployment ID"
// @Success      200 {array} model.SmartServiceDeploymentExtendedParameter
// @Failure      500
// @Failure      401
// @Router       /deployments/{id}/parameters [get]
func (this *Deployments) Parameters(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.GET("/deployments/:id/parameters", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		//TODO: replace with real code
		log.Println(token)
		json.NewEncoder(writer).Encode(model.SmartServiceModel{})
	})
}

// Start godoc
// @Summary      creates a smart-service instance from the deployment
// @Description  creates a smart-service instance from the deployment
// @Tags         deployments, instances
// @Accept       json
// @Produce      json
// @Param        id path string true "Deployment ID"
// @Param        message body model.SmartServiceDeploymentParameters true "SmartServiceDeploymentParameter"
// @Success      200 {object} model.SmartServiceInstance
// @Failure      500
// @Failure      401
// @Router       /deployments/{id}/instances [post]
func (this *Deployments) Start(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.POST("/deployments/:id/instances", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		//TODO: replace with real code
		log.Println(token)
		json.NewEncoder(writer).Encode(model.SmartServiceDeployment{})
	})
}
