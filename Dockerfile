FROM golang as builder

WORKDIR /go/src

ADD . github.com/realitycheck/claps

RUN CGO_ENABLED=0 GOOS=linux go install -a -installsuffix cgo github.com/realitycheck/claps/cmd/claps

FROM scratch

COPY --from=builder /go/bin/claps .

CMD ["./claps"]