# syntax=docker/dockerfile:1

FROM golang:1.26.4-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/goshort .

FROM alpine:3.22

RUN adduser -D -H -u 10001 goshort
USER goshort

COPY --from=build /out/goshort /goshort

EXPOSE 9000

ENTRYPOINT ["/goshort"]
