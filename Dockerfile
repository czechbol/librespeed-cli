FROM alpine:3.17

# Copy librespeedtest binary
COPY librespeedtest /bin/librespeedtest

ENTRYPOINT ["/bin/librespeedtest"]