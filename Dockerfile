# Multi-arch without QEMU: the build stage runs natively on the CI runner
# ($BUILDPLATFORM) and Go cross-compiles for $TARGETARCH. The final stage only
# copies a static binary into distroless, so nothing executes for the target arch.
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS build
ARG TARGETOS TARGETARCH VERSION=dev
WORKDIR /src
COPY go.* ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/app .

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/app /app
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app"]
