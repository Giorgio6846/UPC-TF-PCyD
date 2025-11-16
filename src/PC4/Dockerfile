FROM golang:1.25.3-alpine3.21 AS build

WORKDIR /app
COPY . .

RUN go mod download
RUN go build -o server .

FROM alpine:3.21

WORKDIR /app
COPY --from=build /app/server .
CMD ["./server"]