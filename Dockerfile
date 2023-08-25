ARG GOLANG_VERSION=1.21.0

ARG COMMIT
ARG VERSION

ARG GOOS="linux"
ARG GOARCH="amd64"

FROM docker.io/golang:${GOLANG_VERSION} as build

WORKDIR /vultr-exporter

COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download

COPY cmd/server cmd/server
COPY collector collector

ARG GOOS
ARG GOARCH

ARG VERSION
ARG COMMIT

RUN CGO_ENABLED=0 GOOS=${GOOS} GOARCH=${GOARCH} \
    go build \
    -ldflags "-X main.OSVersion=${VERSION} -X main.GitCommit=${COMMIT}" \
    -a -installsuffix cgo \
    -o /bin/server \
    ./cmd/server


FROM gcr.io/distroless/static

LABEL org.opencontainers.image.source https://github.com/DazWilkin/vultr-exporter

COPY --from=build /bin/server /

ENTRYPOINT ["/server"]
CMD ["--entrypoint=0.0.0.0:8080","--path=/metrics"]
