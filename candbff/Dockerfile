FROM docker.io/library/alpine:latest

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy the pre-built host binary
COPY candbff .

EXPOSE 8080

CMD ["./candbff"]
