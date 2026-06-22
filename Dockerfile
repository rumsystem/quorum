FROM golang:1.25-alpine AS build
RUN addgroup -S quorum && adduser -S -G quorum quorum
RUN apk add --no-cache git
WORKDIR /src
COPY . .
RUN make linux

FROM scratch
WORKDIR /
COPY --from=build /etc/passwd /etc/passwd
COPY --from=build /etc/group /etc/group
COPY --from=build /src/dist/quorum_linux_amd64/quorum /quorum
EXPOSE 8000
EXPOSE 8001
EXPOSE 8002
USER quorum:quorum
ENTRYPOINT ["/quorum"]
