
## OpenAPI
uses https://github.com/swaggo/swag

### installation
```
go install github.com/swaggo/swag/cmd/swag@latest
```

### generating
```
swag init --parseDependency -d ./pkg/api -g api.go
```

### swagger ui
if the config variable UseSwaggerEndpoints is set to true, a swagger ui is accessible on /swagger/index.html (http://localhost:8080/swagger/index.html)