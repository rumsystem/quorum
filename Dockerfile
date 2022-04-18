FROM golang:1.18-alpine AS build
RUN addgroup -S quorum && adduser -S -G quorum quorum
RUN apk add build-base
RUN apk add git
WORKDIR /src
COPY . .
RUN make linux

FROM scratch
WORKDIR /
COPY --from=build /src/dist/linux_amd64/quorum /quorum
EXPOSE 8000
EXPOSE 8001
EXPOSE 8002
USER quorum:quorum
ENTRYPOINT ["/quorum"]
