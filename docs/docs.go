// Package docs GENERATED BY SWAG; DO NOT EDIT
// This file was generated by swaggo/swag
package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "contact": {},
        "license": {
            "name": "Apache 2.0",
            "url": "http://www.apache.org/licenses/LICENSE-2.0.html"
        },
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/": {
            "get": {
                "description": "checks health and reachability of the service",
                "tags": [
                    "health"
                ],
                "summary": "health check",
                "responses": {
                    "200": {
                        "description": ""
                    }
                }
            }
        },
        "/health": {
            "get": {
                "description": "checks health and reachability of the service",
                "tags": [
                    "health"
                ],
                "summary": "health check",
                "responses": {
                    "200": {
                        "description": ""
                    }
                }
            }
        },
        "/instances": {
            "get": {
                "description": "checks health and reachability of the service",
                "tags": [
                    "health"
                ],
                "summary": "returns a list of smart-service isntances",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "type": "array",
                            "items": {
                                "$ref": "#/definitions/model.SmartServiceInstance"
                            }
                        }
                    },
                    "500": {
                        "description": ""
                    }
                }
            }
        }
    },
    "definitions": {
        "model.SmartServiceInstance": {
            "type": "object",
            "properties": {
                "id": {
                    "type": "string"
                }
            }
        }
    },
    "securityDefinitions": {
        "Bearer": {
            "type": "apiKey",
            "name": "Authorization",
            "in": "header"
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "0.1",
	Host:             "localhost:8080",
	BasePath:         "/",
	Schemes:          []string{},
	Title:            "Smart-Service-Repository API",
	Description:      "",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
