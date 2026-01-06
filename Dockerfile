FROM golang:1.24-alpine AS builder

WORKDIR /src
COPY go.mod /src/go.mod
COPY main.go /src/main.go

RUN go build -o /out/alertmanager-teams-adapter /src/main.go

FROM alpine:3.20

WORKDIR /app
COPY --from=builder /out/alertmanager-teams-adapter /app/alertmanager-teams-adapter

EXPOSE 8080
CMD ["/app/alertmanager-teams-adapter"]
