FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /slo-bff .

FROM gcr.io/distroless/static-debian12
COPY --from=build /slo-bff /slo-bff
EXPOSE 9090
ENTRYPOINT ["/slo-bff"]
