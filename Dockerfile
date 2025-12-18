# Start with a Golang and Node.js build image
FROM golang:1.25-alpine AS build

# install node and build tools
# RUN apk update && apk add --no-cache nodejs make build-base && apk add --update npm

WORKDIR /app

# Copy only necessary Go module files
COPY go.mod .
COPY go.sum .

# install templ globally
RUN go install github.com/a-h/templ/cmd/templ@latest

# Download the Go module dependencies
RUN go mod download

# Copy the entire application source code
COPY . .

# Build the Golang application
RUN templ generate
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-extldflags=-static" -o /build ./cmd/server/main.go

# Start a new stage using a lightweight Alpine image
FROM gcr.io/distroless/static-debian11 AS run

# install ghostscript and ffmpeg
RUN apt-get update && apt-get install -y ghostscript ffmpeg

WORKDIR /app

# RUN cd /app
# # Copy the Caddyfile to the appropriate location
# COPY ./Caddyfile /etc/caddy/Caddyfile

# # Copy the built Golang application from the build image
COPY --from=build /build /app
COPY --from=build /app/web /app/web
COPY --from=build /app/internal /app/internal

EXPOSE 8080

ENTRYPOINT ["/app/build"]