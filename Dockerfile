# Build with the golang image
FROM golang:1.14-alpine AS build

# Add git
RUN apk add git

# Set workdir
WORKDIR /app

# Add dependencies
COPY app/go.mod .
COPY app/go.sum .
RUN go mod download

# Build
COPY app .
RUN CGO_ENABLED=0 go build

# Generate final image
FROM scratch
COPY --from=build /app/service-dashboard /service-dashboard
COPY --from=build /app/www               /srv
USER 1000

EXPOSE 8000

ENTRYPOINT [ "/service-dashboard" ]
