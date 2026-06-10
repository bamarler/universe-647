# Sophon backend — static Go binary in a distroless image (~25MB).
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /sophon ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /sophon /sophon
EXPOSE 8080
ENTRYPOINT ["/sophon"]
