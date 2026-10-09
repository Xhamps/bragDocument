FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/bragdoc ./cmd/bragdoc
# EXPORT_DIR in compose: a fresh named volume copies this dir's nonroot ownership.
RUN mkdir -p /out/data/exports

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/bragdoc /bragdoc
COPY --from=build --chown=65532:65532 /out/data /data
EXPOSE 8080
ENTRYPOINT ["/bragdoc"]
CMD ["api"]
