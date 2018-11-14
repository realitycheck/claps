FROM golang as builder

RUN go get -u github.com/golang/dep/cmd/dep
ADD . /go/src/github.com/realitycheck/claps
WORKDIR /go/src/github.com/realitycheck/claps
RUN dep ensure -vendor-only
RUN CGO_ENABLED=0 GOOS=linux go install -a -installsuffix cgo .

FROM scratch

COPY --from=builder /go/bin/claps /
ENV PATH /
STOPSIGNAL SIGTERM
ENTRYPOINT [ "claps" ]