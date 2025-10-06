FROM golang:1.25 AS builder

WORKDIR /app

COPY go.mod ./

COPY . .

RUN ls -la
RUN ls -la server || echo "Pasta server não encontrada!"

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server-bin ./server

FROM alpine:3.19

WORKDIR /app

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/server-bin ./server

ENV PORT=8001
ENV PEERS=""

EXPOSE 8001

CMD ["./server"]