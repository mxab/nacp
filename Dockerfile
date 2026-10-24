FROM alpine:3.24.3 AS certificates
RUN apk add --no-cache ca-certificates

FROM scratch
COPY nacp /nacp
COPY --from=certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
USER 10001:10001
ENTRYPOINT ["/nacp"]
