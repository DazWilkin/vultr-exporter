ARG GOLANG_VERSION=1.24.0

ARG COMMIT
ARG VERSION

ARG TARGETOS
ARG TARGETARCH

FROM --platform=${TARGETARCH} docker.io/golang:${GOLANG_VERSION} AS build

WORKDIR /vultr-exporter

COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download

COPY cmd/server cmd/server
COPY collector collector

ARG TARGETOS
ARG TARGETARCH

ARG VERSION
ARG COMMIT

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build \
    -ldflags "-X main.OSVersion=${VERSION} -X main.GitCommit=${COMMIT}" \
    -a -installsuffix cgo \
    -o /bin/server \
    ./cmd/server


FROM --platform=${TARGETARCH} gcr.io/distroless/static-debian12:latest

LABEL org.opencontainers.image.source=https://github.com/DazWilkin/vultr-exporter

COPY --from=build /bin/server /

ENTRYPOINT ["/server"]
CMD ["--entrypoint=0.0.0.0:8080","--path=/metrics"]
