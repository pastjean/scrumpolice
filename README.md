# Scrum Police

Scrum bot ask every member of your team for a scrum report then reports it to
your team's channel at a predefined time.

- [Jason Fried's - Status meetings are the scourge](https://m.signalvnoise.com/status-meetings-are-the-scourge-39f49267ca90) started all the fuzz.


# Installation

...

# Development

Have a working go environment (since 1.8 just install go)

Clone the repo in your workspace:

```sh
git clone git@github.com:scrumpolice/scrumpolice.git $HOME/go/src/github.com/scrumpolice/scrumpolice
```

Get dependencies:

```sh
go get -u ./...
```

Run it:

```sh
go run cmd/scrumpolice/scrumpolice.go
```
