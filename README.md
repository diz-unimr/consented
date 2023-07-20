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
