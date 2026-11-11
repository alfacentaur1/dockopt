# syntax=docker/dockerfile:1

# dockopt in a container, for CI systems that prefer images to binaries:
#   docker run --rm -v "$PWD:/src" -w /src dockopt analyze
# (bench needs the docker CLI and socket, so run that one on the host)

FROM golang:1.27 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/dockopt ./cmd/dockopt

FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/dockopt /usr/local/bin/dockopt
ENTRYPOINT ["/usr/local/bin/dockopt"]
