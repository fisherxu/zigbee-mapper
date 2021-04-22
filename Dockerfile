FROM golang:1.12-alpine3.10
RUN apk add gcc libc-dev
RUN mkdir -p $GOPATH/src/zigbee-mapper
COPY . $GOPATH/src/zigbee-mapper
RUN CGO_ENABLED=1 go install $GOPATH/src/zigbee-mapper/zigbee-mapper
ENTRYPOINT ["zigbee-mapper"]
