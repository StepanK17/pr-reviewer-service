FROM golang:1.24.4-alpine

WORKDIR /tests

RUN apk add --no-cache curl

COPY e2e_tests/go.mod e2e_tests/go.sum ./
RUN go mod download

COPY e2e_tests/ ./

CMD ["go", "test", "-v", "./..."]