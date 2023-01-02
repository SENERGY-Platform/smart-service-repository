<a href="https://github.com/SENERGY-Platform/smart-service-repository/.github/workflows/tests.yml" rel="nofollow">
    <img src="https://github.com/SENERGY-Platform/smart-service-repository/.github/workflows/tests.yml/badge.svg" alt="Tests" />
</a>

## Parameter
parameters returned by `GET /releases/:id/parameters` depend on form fields defined on the start-event element in the design process.
those form fields can have the following properties

examples can be found in ./pkg/tests/resources

### description

- property name: `description`
- description: sets the description of the parameter
- value: string
- value example: `some discription text`

### iot

- property name: `iot`
- description: creates options of selectable IoT entities (Devices, Device-Groups, Imports), depending on the field value und the `criteria` or `criteria_list` property
- value: comma seperated list of `device`, `group` and `import`
- value example: `device,group`

### criteria

- property name: `criteria`
- description: filters options for `iot` property
- value: `json.Marshal(Criteria{})`
- value example: `{"interaction": "request, "function_id": "urn:infai:ses:measuring-function:826e5a04-71cc-4935-9fd4-92c930dc06bb"}`

### criteria_list

- property name: `criteria_list`
- description: filters options for `iot` property
- value: `json.Marshal([]Criteria{})`
- value example: `[{"interaction": "request, "function_id": "urn:infai:ses:measuring-function:826e5a04-71cc-4935-9fd4-92c930dc06bb"}]`


### entity_only

- property name: `entity_only`
- description: options for `iot` property contain only the entity (device-id etc) without service-id or path. useful if option is intented to be used with multiple different functions/services
- value: boolean
- value example: `true`

### same_entity

- property name: `same_entity`
- description: references other form-field to indicate, that options in this form field are only valid if they have the same `entity_id` as the option selected in the referenced form field. useful if parameter selections depend on each other, for example if multiple different services should be selected for the same device
- value: id of other form field

### options

- property name: `options`
- description: if options are not defined by the `iot` property, it is possible to set them manually
- value: `json.Marshal(map[string]interface{})`
- value example: `{"label": "value", "the solution": 42}`

### order

- property name: `order`
- description: used to define in which order parameters should be returned
- value: number
- value example: `0`

### multiple

- property name: `multiple`
- description: informs user, that a list of values is expected
- value: boolean
- value example: `true`

### auto_select_all

- property name: `auto_select_all`
- description: parameter will not be displayed to user. instead the parameter becomes a list of all parameter options. must be used in combination with `multiple`. can be used to select all devices of a user that matches the given criteria.
- value: boolean
- value example: `true`

### characteristic_id

- property name: `characteristic_id`
- description: sets Characteristc to SmartServiceExtendedParameter as a hint in the result of `GET /releases/:id/parameters` what the data-structure and semantic meaning of the expected json value is.
- value: string
- value example: `urn:infai:ses:characteristic:5b4eea52-e8e5-4e80-9455-0382f81a1b43`

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