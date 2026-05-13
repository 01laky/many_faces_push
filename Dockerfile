# Skeleton Dockerfile — builds the placeholder binary only. Replace with distroless + nonroot when gRPC ships.
FROM golang:1.23-bookworm AS build

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/push-worker ./cmd/push-worker

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /

COPY --from=build /out/push-worker /push-worker

EXPOSE 50053

USER nonroot:nonroot

ENTRYPOINT ["/push-worker"]
