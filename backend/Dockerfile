FROM golang:1.22-alpine AS build

WORKDIR /src
COPY go.mod ./
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/seed ./scripts

FROM alpine:3.20
RUN apk add --no-cache ca-certificates && adduser -D -H app
COPY --from=build /out/server /usr/local/bin/server
COPY --from=build /out/seed /usr/local/bin/seed
USER app
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/server"]
