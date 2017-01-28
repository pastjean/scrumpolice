```                                           _ _
 ___  ___ _ __ _   _ _ __ ___  _ __   ___ | (_) ___ ___
/ __|/ __| '__| | | | '_ ` _ \| '_ \ / _ \| | |/ __/ _ \
\__ \ (__| |  | |_| | | | | | | |_) | (_) | | | (_|  __/
|___/\___|_|   \__,_|_| |_| |_| .__/ \___/|_|_|\___\___|
                              |_|
```

[![Build Status](https://travis-ci.org/scrumpolice/scrumpolice.svg?branch=master)](https://travis-ci.org/scrumpolice/scrumpolice)
[![Go Report Card](https://goreportcard.com/badge/github.com/scrumpolice/scrumpolice)](https://goreportcard.com/report/github.com/scrumpolice/scrumpolice)

Scrum bot ask every member of your team for a scrum report then reports it to
your team's channel at a predefined time.

- [Jason Fried's - Status meetings are the scourge](https://m.signalvnoise.com/status-meetings-are-the-scourge-39f49267ca90) started all the fuzz.

# Installation 

```sh 
# TODO
```

# Development

Have a working go environment (since 1.8 just install go) otherwise you need the
`$GOPATH` set and use that instead of `$HOME/go`.

```sh
go get -u -d github.com/scrumpolice/scrumpolice
cd $HOME/go/src/github.com/scrumpolice/scrumpolice
# Run it
go run cmd/scrumpolice/scrumpolice.go
```
