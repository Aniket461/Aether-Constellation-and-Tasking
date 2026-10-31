FROM golang:1.27.3-bookworm AS build
WORKDIR /src
COPY constellation/go.mod constellation/go.sum constellation/
COPY tasking/backend/go.mod tasking/backend/go.sum tasking/backend/
WORKDIR /src/tasking/backend
RUN go mod download
WORKDIR /src
COPY constellation constellation
COPY tasking/backend tasking/backend
WORKDIR /src/tasking/backend
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=build /out/api /api
EXPOSE 8080
USER nobody
ENTRYPOINT ["/api"]
