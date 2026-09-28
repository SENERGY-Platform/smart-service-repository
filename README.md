<a href="https://github.com/SENERGY-Platform/smart-service-repository/actions/workflows/tests.yml" rel="nofollow">
    <img src="https://github.com/SENERGY-Platform/smart-service-repository/actions/workflows/tests.yml/badge.svg?branch=main" alt="Tests" />
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

### aspects in a criteria

A criteria names its aspects in `aspect_ids`:

```json
{"function_id": "urn:infai:ses:measuring-function:...", "aspect_ids": ["urn:infai:ses:aspect:a", "urn:infai:ses:aspect:b"]}
```

Several aspects in **one** criteria are an AND on a single content variable: the same variable
has to carry all of them, each of them covering its own aspect subtree. That is a different
AND from the one over `criteria_list`, where each entry may be answered by a different
variable of the device type. Two sibling aspects in one criteria therefore match nothing,
because a content variable carries at most one aspect out of a classified hierarchy.

`aspect_id` is deprecated and kept as the alias for a list with a single element, so a design
written before the lists keeps its results. Both spellings are passed on to device-selection as
the design wrote them; it resolves the alias.


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

## OpenTelemetry

Traces are sent to an OTLP collector over gRPC. The endpoint is set with the config value
`otel_endpoint` (env `OTEL_ENDPOINT`); if it is empty, the default
`jaeger.logging.svc.cluster.local:4317` is used.

What is instrumented:

- incoming requests, through `otelx.HTTPOpenTelemetry` of
  [gin-middleware](https://github.com/SENERGY-Platform/gin-middleware/tree/main/otelx). It picks up
  the trace-context of the caller and puts `user_id` and `username` into the baggage.
- mongodb, through `otelmongo`.
- outgoing requests to camunda, device-selection, permissions-v2, the notifier and the delete-urls
  of modules. They carry trace-context and baggage on, so a request can be followed across services.
- log records, through the open-telemetry handler of `struct-logger`. It writes the baggage into the
  record and the record into the current span, but **only for the context based log methods**
  (`ErrorContext`, `InfoContext`, ...). A `logger.Error()` without a context stays untraced.

`POST /releases/:id/instances` adds the id of the new smart-service-instance to the baggage as
`smart_service_instance_id` as soon as it is known, so every following request and log record of
that creation carries it.

Not propagated: the keycloak token-exchange and the device-repository client, which has no
context aware methods.

## MongoDB configuration

| Env var | Default | Notes |
|---|---|---|
| `MONGO_URL` | `mongodb://localhost:27017` | Full connection string including scheme, passed to the driver unchanged; must not contain credentials. |
| `MONGO_USER` | empty | No authentication when empty. |
| `MONGO_PASSWORD` | empty | Required when `MONGO_USER` is set; never printed at startup and masked when the config is formatted or marshalled. |
| `MONGO_AUTH_SOURCE` | `admin` | Database the user is defined in. |
| `MONGO_DATABASE` | `smart_service_repository` | Must not be empty; collections are `MONGO_COLLECTION_DESIGN`, `MONGO_COLLECTION_RELEASE`, `MONGO_COLLECTION_INSTANCE`, `MONGO_COLLECTION_MODULE` and `MONGO_COLLECTION_VARIABLES`. |

In `config.json` these are `mongo_url`, `mongo_user`, `mongo_password`, `mongo_auth_source` and `mongo_database`.

When `MONGO_USER` is set, the credentials are built from `MONGO_USER`, `MONGO_PASSWORD` and `MONGO_AUTH_SOURCE` alone: they replace any user, password, `authSource` and `authMechanism` given in `MONGO_URL`.

Every applied environment variable is printed at startup, `MONGO_URL` included, so credentials belong in `MONGO_USER`/`MONGO_PASSWORD`, never in `MONGO_URL`. Startup fails unless an authenticated `listCollections` on `MONGO_DATABASE` succeeds within 10 seconds.

### Tests

`TestStartAuthenticates` (`pkg/database/mongo`) runs only without `-short` and when `MONGO_AUTH_TEST_URL`, `MONGO_AUTH_TEST_USER` and `MONGO_AUTH_TEST_PASSWORD` are set. The user and password are root credentials of a throwaway server with access control; the test creates and removes its own users and databases there. Example:

    docker run -d --rm --name smart-service-repository-auth-test -p 127.0.0.1:27018:27017 -e MONGO_INITDB_ROOT_USERNAME=root -e MONGO_INITDB_ROOT_PASSWORD=rootpw mongo:8.0
    MONGO_AUTH_TEST_URL=mongodb://127.0.0.1:27018 MONGO_AUTH_TEST_USER=root MONGO_AUTH_TEST_PASSWORD=rootpw go test ./pkg/database/mongo/
    docker stop smart-service-repository-auth-test

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