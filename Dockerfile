FROM balenalib/raspberry-pi-alpine-golang as go-builder

RUN apk --update add git make npm

WORKDIR /src/AdGuardHome
COPY . /src/AdGuardHome
RUN make

#####################################################################

FROM resin/rpi-alpine
#FROM resin/raspberry-pi-alpine
LABEL maintainer="Erik Rogers <erik.rogers@live.com>"

RUN apk --no-cache --update add ca-certificates

WORKDIR /root/
COPY --from=go-builder /src/AdGuardHome/AdGuardHome /AdGuardHome
COPY --from=go-builder /src/AdGuardHome/AdGuardHome.yaml /AdGuardHome.yaml

EXPOSE 53 3000

VOLUME /data

#ENTRYPOINT ["/AdGuardHome"]
ENTRYPOINT ["/bin/sh"]
#CMD ["-h", "0.0.0.0"]

