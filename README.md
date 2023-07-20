# consented
![go](https://github.com/diz-unimr/consented/actions/workflows/build.yml/badge.svg) ![docker](https://github.com/diz-unimr/consent-to-fhir/actions/workflows/release.yml/badge.svg) [![codecov](https://codecov.io/github/diz-unimr/consented/branch/main/graph/badge.svg?token=4ciJIXKAK5)](https://codecov.io/github/diz-unimr/consented)
> REST service to query consent status information via gICS

This service currently supports querying a single policy status by patient ID, domain and (optionally) date.
It uses the [$isConsented](https://www.ths-greifswald.de/wp-content/uploads/tools/fhirgw/ig/2023-1-0/ImplementationGuide-markdown-Einwilligungsmanagement-Operations-isConsented.html) operation of the gICS TTP FHIR Gateway API to query predefined policies with the given 
input data.


## RESTful API

<details>
 <summary><code>GET</code> <code><b>/consent/status/{patientId}/{domain}</b></code> <code>get consent status by patient ID and domain</code></summary>

##### Path parameters

> | name        |  type     | data type | description        |
> |-------------|-----------|-----------|--------------------|
> | `patientId` |  required | string    | The gICS signer ID |
> | `domain`    |  required | string    | The gICS domain    |

##### Query parameters

> | name   | type     | data type       | description               |
> |--------|----------|-----------------|---------------------------|
> | `date` | optional | date (YY-MM-DD) | Date to resolve status to |

##### Responses

> | http code | content-type       | response                                              |
> |-----------|--------------------|-------------------------------------------------------|
> | `200`     | `application/json` | `{"consented": [true\|false], "domain": "[domain]" }` |
> | `400`     | `application/json` | `{"error": "[error string]"}`                         |
> | `401`     |                    |                                                       |
> | `404`     | `application/json` | `{"error": "[error string]"}`                          |
> | `502`     | `application/json` | `{"error": "[error string]"}`                          |

##### Example cURL

> ```bash
>  curl -X GET -H "Content-Type: application/json" https://localhost/consent/status/42/MII
> ```


#### Example response

>```json
>{
>  "consented": true,
>  "domain": "MII"
>}
>```
</details>

## Configuration properties

| Name                      | Default         | Description                             |
|---------------------------|-----------------|-----------------------------------------|
| `app.name`                | consent-to-fhir | Application name                        |
| `app.log-level`           | info            | Log level (error,warn,info,debug,trace) |
| `app.http.auth.user`      |                 | HTTP endpoint Basic Auth user           |
| `app.http.auth.password`  |                 | HTTP endpoint Basic Auth password       |
| `app.http.port`           | 8080            | HTTP endpoint port                      |
| `gics.signer-id`          | Patienten-ID    | Target consent signerId                 |
| `gics.fhir.base`          |                 | TTP-FHIR base url                       |
| `gics.fhir.auth.user`     |                 | TTP-FHIR Basic auth user                |
| `gics.fhir.auth.password` |                 | TTP-FHIR Basic auth password            |


### Environment variables

Override configuration properties by providing environment variables with their respective names.
Upper case env variables are supported as well as underscores (`_`) instead of `.` and `-`.


# Deployment

Example via `docker compose`:
```yml
consent-to-fhir:
    image: ghcr.io/diz-unimr/consented:latest
    restart: unless-stopped
    environment:
      APP_NAME: consented
      APP_LOG_LEVEL: info
      APP_HTTP_AUTH_USER: test
      APP_HTTP_AUTH_PASSWORD: test
      APP_HTTP_PORT: 8080
      GICS_SIGNER_ID: Patienten-ID
      GICS_FHIR_BASE: https://gics.local/ttp-fhir/fhir/gics/
      GICS_FHIR_AUTH_USER: test
      GICS_FHIR_AUTH_PASSWORD: test
```

# License

[AGPL-3.0](https://www.gnu.org/licenses/agpl-3.0.en.html)
