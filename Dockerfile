FROM golang:1.20.3-alpine as builder

RUN apk add --no-cache bash upx

# Set working directory
WORKDIR /usr/src/librespeedtest

# Copy librespeedtest
COPY . .

# Build librespeedtest
RUN ./build.sh

FROM alpine:3.17

# Copy librespeedtest binary
COPY --from=builder /usr/src/librespeedtest/out/librespeedtest* /bin/librespeedtest

CMD ["/bin/librespeedtest"]
