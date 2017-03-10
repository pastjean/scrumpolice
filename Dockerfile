FROM scratch

COPY scrumpolice /scrumpolice

ENTRYPOINT ["/scrumpolice"]