FROM golang:alpine AS build

RUN apk add --update git make build-base npm && \
    rm -rf /var/cache/apk/*

WORKDIR /src/AdGuardHome
COPY . /src/AdGuardHome
RUN make

FROM alpine:latest
LABEL maintainer="AdGuard Team <devteam@adguard.com>"

# Update CA certs
RUN apk --no-cache --update add ca-certificates && \
    rm -rf /var/cache/apk/*

COPY --from=build /src/AdGuardHome/AdGuardHome /AdGuardHome

EXPOSE 53 3000

VOLUME /data

ENTRYPOINT ["/AdGuardHome"]
CMD ["-h", "0.0.0.0"]