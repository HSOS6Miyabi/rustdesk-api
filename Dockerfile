FROM --platform=${TARGETPLATFORM} golang:1.23-alpine AS build
WORKDIR /source
RUN apk add --no-cache build-base git python3
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN python3 scripts/package_geoip.py && python3 scripts/package_geoip.py --verify
RUN CGO_ENABLED=1 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/apimain ./cmd/apimain.go

FROM alpine:3.21
RUN apk add --no-cache tzdata ca-certificates
WORKDIR /app
COPY --from=build /out/apimain /app/apimain
COPY --from=build /source/conf /app/conf
COPY --from=build /source/docs /app/docs
COPY --from=build /source/resources /app/resources
RUN mkdir -p /app/data /app/runtime
VOLUME /app/data
EXPOSE 21114
CMD ["./apimain"]
