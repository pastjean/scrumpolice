FROM golang:alpine as builder

# Install SSL ca certificates
RUN apk update && apk add git ca-certificates tzdata curl
RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
# Create appuser
RUN adduser -D -g '' appuser
 
WORKDIR /go/src/github.com/pastjean/scrumpolice/
COPY . .

RUN go get ./...
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o /scrumpolice .

FROM alpine:3.6
# We would like to use "scratch" but the tzdata package is kinda complicated
RUN apk add --no-cache ca-certificates tzdata

# COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /scrumpolice /scrumpolice

USER appuser

ENTRYPOINT ["/scrumpolice"]
