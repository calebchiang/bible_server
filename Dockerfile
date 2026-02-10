FROM golang:1.22-bookworm

# Install SQLite dependencies
RUN apt-get update && apt-get install -y gcc sqlite3 libsqlite3-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Enable CGO
ENV CGO_ENABLED=1

RUN go build -o app .

CMD ["./app"]
