FROM golang:1.21-alpine3.18 AS build

WORKDIR /app
COPY go.* ./
RUN go mod download

COPY . .
RUN go get -d -v
RUN GOOS=linux GOARCH=amd64 go build -v

FROM alpine:3.18 as run

RUN apk add --no-cache tzdata
ENV TZ=Europe/Berlin

WORKDIR /app/
COPY --from=build /app/consented .
COPY --from=build /app/app.yml .
ENV GIN_MODE=release

EXPOSE 8080

ENTRYPOINT ["/app/consented"]
