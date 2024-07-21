FROM golang:1.22-alpine AS build

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o myapp .

FROM scratch

WORKDIR /app

COPY --from=build /app/myapp .
COPY static/ .
RUN 'mkdir -p /app/data/'

EXPOSE 8080
ENTRYPOINT ["/app/myapp"]