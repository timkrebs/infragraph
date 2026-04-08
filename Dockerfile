# Minimal container for InfraGraph.
# GoReleaser pre-builds the binary and copies it into this image.
FROM alpine:3.21 AS base

# Install ca-certificates for TLS connections and tzdata for timezone support.
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user.
RUN addgroup -S infragraph && adduser -S -G infragraph infragraph

FROM scratch

COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=base /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=base /etc/passwd /etc/passwd
COPY --from=base /etc/group /etc/group

# GoReleaser copies the pre-built binary here.
COPY infragraph /usr/bin/infragraph

USER infragraph

EXPOSE 8080

ENTRYPOINT ["infragraph"]
