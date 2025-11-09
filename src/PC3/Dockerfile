FROM golang:1.25.3-alpine3.21 AS build

WORKDIR /app
COPY . .
WORKDIR /app/src

RUN go mod download
RUN go build -o server ./...

FROM alpine:3.21

WORKDIR /app
COPY --from=build /app/src .
CMD ["./server"]