# consented
![go](https://github.com/diz-unimr/consented/actions/workflows/build.yml/badge.svg) ![docker](https://github.com/diz-unimr/consent-to-fhir/actions/workflows/release.yml/badge.svg) [![codecov](https://codecov.io/github/diz-unimr/consented/branch/main/graph/badge.svg?token=4ciJIXKAK5)](https://codecov.io/github/diz-unimr/consented)
> REST service to query consent status information via gICS

This service provides a single endpoint to query consent status information for a patient across all configured gICS domains.

It uses the [$currentPolicyStatesForPerson](https://www.ths-greifswald.de/wp-content/uploads/tools/fhirgw/ig/2.2.0/ImplementationGuide-markdown-Einwilligungsmanagement-Operations-currentPolicyStatesForPerson.html)
operation of the gICS TTP FHIR Gateway API to query policies and provide detailed consent status information for a patient and each configured domain.  

## Background

This service is intended to provide all information needed to decide if a patient should be asked for a consent at the time. 
Detailed policy status information is available, too, and can be used to give feedback to the patient.

## Domain configuration (gICS)

For a domain to be used in consent evaluation, it must be configured via external properties.
The `checkPolicy` property is the only mandatory one that needs to be set. It should be set to the name of the 
policy that is used to determine if a consent should be considered _accepted_.  

Example for the MII Broad consent:

```sh
checkPolicy=MDAT_erheben
```

Additionally, the `departments` property can be set to filter domains and only include them if explicitly requested. 
The property value can be a single value or a list of comma seperated strings.

```sh
departments=Department1,Department2
```

Data from those domains are part of the response if at least one of its values matches a value of the
`departments` property of the HTTP request body:

```json
{
  "departments": ["Department1"]
}
```

The body is optional in the request, though. In case it's missing only domains without the `departments` property set
are considered.

### Caching

Domain information is cached by the service initially on start and periodically via the `gics.update-interval`
application property.


## RESTful API

<details>
 <summary><code>POST</code> <code><b>/consent/status/{patientId}</b></code> <code>get consent status by patient ID</code></summary>

##### Request

###### Path parameter

> | name        |  type     | data type | description        |
> |-------------|-----------|-----------|--------------------|
> | `patientId` |  required | string    | The gICS signer ID |

###### Body

_The body is optional!_

> | content-type       | value                      | description                                   |
> |--------------------|----------------------------|-----------------------------------------------|
> | `application/json` | `{"departments": ["..."]}` | Include listed departments in status response |

##### Responses

_Response JSON interface definitions below._

> | http code | content-type       | response                         |
> |-----------|--------------------|----------------------------------|
> | `200`     | `application/json` | Array of `Consent domain status` |
> | `400`     | `application/json` | `Error`                          |
> | `401`     |                    |                                  |
> | `404`     | `application/json` | `Error`                          |
> | `502`     | `application/json` | `Error`                          |

###### JSON response interfaces

`Consent domain status`

_See `Policy` response below._

| property     | description                       | type                                                                 |
|--------------|-----------------------------------|----------------------------------------------------------------------|
| domain       | domain name                       | `string`                                                             |
| description  | domain description                | `string`                                                             |
| status       | consent status (of `checkPolicy`) | `string` ("accepted", "declined", "expired","withdrawn","not-asked") |
| last-updated | date of last update               | `string` (ISO 8601 date)                                             |
| ask-consent  | patient can be asked for consent  | `boolean`                                                            |
| policies     | domain name                       | Array of `Policy`                                                    |

⚠️ **NOTE**: `ask-consent` _can_ evaluate to `true`, in case a valid consent exists that expires in less than a year.

`Policy`

| property | description   | type      |
|----------|---------------|-----------|
| name     | policy name   | `string`  |
| permit   | policy status | `boolean` |

`Error`

| property | description         | type     |
|----------|---------------------|----------|
| error    | error response text | `string` |

##### Example cURL

> ```bash
>  curl -X POST -H "Content-Type: application/json" https://localhost/consent/status/42
> ```


#### Example response

>```json
>[
>    {
>      "domain": "MII",
>      "description": "Broad Consent",
>      "status": "declined",
>      "last-updated": "2023-09-21T14:13:25.999+02:00",
>      "ask-consent": false,
>      "policies": [
>        {
>          "name": "Erfassung neuer identifizierender Daten (IDAT)",
>          "permit": false
>        },
>        {
>          "name": "Rekontaktierung bezüglich Zusatzbefund im Rahmen der am Standort dafür entwickelten Prozesse und der im Nutzungsantrag angegebenen Bedingungen",
>          "permit": false
>        },
>        {
>          "name": "Erfassung medizinischer Daten (MDAT)",
>          "permit": false
>        }
>      ]
>    },
>    {
>      "domain": "Test",
>      "description": "Test consent",
>      "status": "not-asked",
>      "last-updated": null,
>      "expires": null,
>      "ask-consent": true,
>      "policies": []
>    }
>]
>```
</details>

## Configuration properties

| Name                      | Default   | Description                              |
|---------------------------|-----------|------------------------------------------|
| `app.name`                | consented | Application name                         |
| `app.log-level`           | info      | Log level (error,warn,info,debug,trace)  |
| `app.http.auth.user`      |           | HTTP endpoint Basic Auth user            |
| `app.http.auth.password`  |           | HTTP endpoint Basic Auth password        |
| `app.http.port`           | 8080      | HTTP endpoint port                       |
| `gics.update-interval`    | 30m       | Interval to update domain data from gICS |
| `gics.fhir.base`          |           | TTP-FHIR base url                        |
| `gics.fhir.auth.user`     |           | TTP-FHIR Basic auth user                 |
| `gics.fhir.auth.password` |           | TTP-FHIR Basic auth password             |


### Environment variables

Override configuration properties by providing environment variables with their respective names.
Upper case env variables are supported as well as underscores (`_`) instead of `.` and `-`.


# Deployment

Example via `docker compose`:
```yml
consented:
    image: ghcr.io/diz-unimr/consented:latest
    restart: unless-stopped
    environment:
      APP_NAME: consented
      APP_LOG_LEVEL: info
      APP_HTTP_AUTH_USER: test
      APP_HTTP_AUTH_PASSWORD: test
      APP_HTTP_PORT: 8080
      GICS_UPDATE_INTERVAL: 10m
      GICS_FHIR_BASE: https://gics.local/ttp-fhir/fhir/gics/
      GICS_FHIR_AUTH_USER: test
      GICS_FHIR_AUTH_PASSWORD: test
```

# License

[AGPL-3.0](https://www.gnu.org/licenses/agpl-3.0.en.html)
