FROM golang:1.26 AS builder
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/task-api .

FROM debian:bookworm-slim
COPY --from=builder /bin/task-api /usr/local/bin/task-api
EXPOSE 8080
CMD ["task-api"]
