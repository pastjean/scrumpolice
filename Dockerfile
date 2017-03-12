FROM alpine:3.4
RUN apk add --no-cache ca-certificates tzdata
COPY scrumpolice /scrumpolice

ENTRYPOINT ["/scrumpolice"]