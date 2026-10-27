FROM golang:1.26.5-alpine AS build
WORKDIR /src
ENV CGO_ENABLED=0
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown
RUN go build -trimpath -ldflags="-s -w -X roadmates/backend/internal/buildinfo.Version=${VERSION} -X roadmates/backend/internal/buildinfo.Commit=${COMMIT} -X roadmates/backend/internal/buildinfo.BuildTime=${BUILD_TIME}" -o /out/server ./cmd/server

FROM build AS test
RUN apk add --no-cache gcc musl-dev
ENV CGO_ENABLED=1
RUN go test -race ./... && go vet ./...

FROM alpine:3.23.3 AS runtime
RUN apk add --no-cache ca-certificates && addgroup -S app && adduser -S -G app app
COPY --from=build /out/server /server
USER app
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=3 CMD wget -q -O /dev/null http://127.0.0.1:8080/healthz || exit 1
ENTRYPOINT ["/server"]
