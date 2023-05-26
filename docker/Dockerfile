FROM amd64/alpine:3.16
RUN apk update && apk add bash iproute2 libc6-compat curl
COPY --chmod=700 skyhook/skyhook.elf /sbin/skyhook
VOLUME ["/config"]
EXPOSE 443/tcp 65000/tcp
ENTRYPOINT skyhook server run -c /config/config.yml
