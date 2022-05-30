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
	endpoints = append(endpoints, &Releases{})
}

type Releases struct{}

// List godoc
// @Summary      returns a list of smart-service releases
// @Description  returns a list of smart-service releases
// @Tags         releases
// @Produce      json
// @Success      200 {array} model.SmartServiceRelease
// @Failure      500
// @Failure      401
// @Router       /releases [get]
func (this *Releases) List(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.GET("/releases", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		//TODO: replace with real code
		log.Println(token)
		json.NewEncoder(writer).Encode([]model.SmartServiceRelease{})
	})
}

// Get godoc
// @Summary      returns a smart-service release
// @Description  returns a smart-service release
// @Tags         releases
// @Produce      json
// @Param        id path string true "Release ID"
// @Success      200 {object} model.SmartServiceRelease
// @Failure      500
// @Failure      401
// @Router       /releases/{id} [get]
func (this *Releases) Get(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.GET("/releases/:id", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		//TODO: replace with real code
		log.Println(token)
		json.NewEncoder(writer).Encode(model.SmartServiceRelease{})
	})
}

// Update godoc
// @Summary      updates a smart-service release
// @Description  updates a smart-service release
// @Tags         releases
// @Accept       json
// @Produce      json
// @Param        id path string true "Release ID"
// @Param        message body model.SmartServiceRelease true "SmartServiceRelease"
// @Success      200 {object} model.SmartServiceRelease
// @Failure      500
// @Failure      401
// @Router       /releases/{id} [put]
func (this *Releases) Update(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.PUT("/releases/:id", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		//TODO: replace with real code
		log.Println(token)
		json.NewEncoder(writer).Encode(model.SmartServiceRelease{})
	})
}

// Create godoc
// @Summary      create a smart-service release
// @Description  creates a smart-service release
// @Tags         releases
// @Accept       json
// @Produce      json
// @Param        id path string true "Release ID"
// @Param        message body model.SmartServiceRelease true "SmartServiceRelease"
// @Success      200 {object} model.SmartServiceRelease
// @Failure      500
// @Failure      401
// @Router       /releases [post]
func (this *Releases) Create(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.POST("/releases", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		//TODO: replace with real code
		log.Println(token)
		json.NewEncoder(writer).Encode(model.SmartServiceRelease{})
	})
}

// Delete godoc
// @Summary      removes a smart-service release
// @Description  removes a smart-service release
// @Tags         releases
// @Accept       json
// @Produce      json
// @Param        id path string true "Release ID"
// @Success      200
// @Failure      500
// @Failure      401
// @Router       /releases/{id} [delete]
func (this *Releases) Delete(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.POST("/releases/:id", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		//TODO: replace with real code
		log.Println(token)
		json.NewEncoder(writer).Encode(model.SmartServiceRelease{})
	})
}

// Parameters godoc
// @Summary      returns parameters of a release
// @Description  returns parameters of a release
// @Tags         releases, parameter
// @Produce      json
// @Param        id path string true "Release ID"
// @Success      200 {array} model.SmartServiceExtendedParameter
// @Failure      500
// @Failure      401
// @Router       /releases/{id}/parameters [get]
func (this *Releases) Parameters(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.GET("/releases/:id/parameters", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		//TODO: replace with real code
		log.Println(token)
		json.NewEncoder(writer).Encode(model.SmartServiceDesign{})
	})
}

// Start godoc
// @Summary      creates a smart-service instance from the release
// @Description  creates a smart-service instance from the release
// @Tags         releases, instances
// @Accept       json
// @Produce      json
// @Param        id path string true "Release ID"
// @Param        message body model.SmartServiceParameters true "SmartServiceParameter"
// @Success      200 {object} model.SmartServiceInstance
// @Failure      500
// @Failure      401
// @Router       /releases/{id}/instances [post]
func (this *Releases) Start(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.POST("/releases/:id/instances", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		//TODO: replace with real code
		log.Println(token)
		json.NewEncoder(writer).Encode(model.SmartServiceRelease{})
	})
}
