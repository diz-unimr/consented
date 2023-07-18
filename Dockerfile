FROM golang:1.19-alpine3.16 AS build

WORKDIR /app
COPY go.* ./
RUN go mod download

COPY . .
RUN go get -d -v
RUN GOOS=linux GOARCH=amd64 go build -v

FROM alpine:3.16 as run

WORKDIR /app/
COPY --from=build /app/consented .
COPY --from=build /app/app.yml .
ENV GIN_MODE=release

EXPOSE 8080

ENTRYPOINT ["/app/consented"]
