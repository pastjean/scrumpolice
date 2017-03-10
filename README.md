```
                                           _ _
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

# Usage 


Create your configuration file: 

```json
{
  "teams": [
    {
      "channel": "themostaswesometeamchannel",
      "name": "L337 team",
      "members": [
        "@fboutin2",
        "@lbourdages",
        "@pastjean"
      ],
      "question_sets": [
        {
          "questions": [
            "What did you do yesterday?",
            "What will you do today?",
            "Are you being blocked by someone for a review? who ? why ?",
            "How will you dominate the world"
          ],
          "report_schedule_cron": "0 0 8 * * 1-5",
          "first_reminder_limit": "+55m",
          "last_reminder_limit": "+65m"
        }
      ]
    }
  ]
}
```

Run the bot with a slack bot user token

```sh
SCRUMPOLICE_SLACK_TOKEN=xoxb-mytoken scrumpolice -config config.json
```

# Development

Have a working go environment (since 1.8 just install go) otherwise you need the
`$GOPATH` set and use that instead of `$HOME/go`.

```sh
go get github.com/scrumpolice/scrumpolice
cd $HOME/go/src/github.com/scrumpolice/scrumpolice
dep ensure
# Run it
SCRUMPOLICE_SLACK_TOKEN=xoxb-mytoken go run cmd/scrumpolice/scrumpolice.go -config config.example.json
```
