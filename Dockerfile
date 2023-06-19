FROM golang:1.20

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . ./

# COPY .env ./

RUN go build -o chat-backend

EXPOSE 8080

CMD ["./chat-backend"]