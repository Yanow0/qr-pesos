############################
# STEP 1 build executable binary
############################
FROM golang:alpine as builder
# Install git + SSL ca certificates.
# Ca-certificates is required to call HTTPS endpoints.
RUN apk update && apk add --no-cache git ca-certificates gcc g++ make && update-ca-certificates
WORKDIR /usr/src/app
COPY . .
RUN go mod download
RUN go mod verify
#  Build the binary static link
WORKDIR /usr/src/app
RUN go build -a -ldflags "-linkmode external -extldflags '-static' -s -w" -o qr-pesos
############################
# STEP 2 build a small image
############################
FROM alpine
WORKDIR /
# Import from builder.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
# Copy our static executable and resources
COPY --from=builder /usr/src/app/qr-pesos /qr-pesos
COPY --from=builder /usr/src/app/static /static
COPY --from=builder /usr/src/app/templates /templates
# Use an unprivileged user.
USER root
ENTRYPOINT ["./qr-pesos"]