```
                                           _ _
 ___  ___ _ __ _   _ _ __ ___  _ __   ___ | (_) ___ ___
/ __|/ __| '__| | | | '_ ` _ \| '_ \ / _ \| | |/ __/ _ \
\__ \ (__| |  | |_| | | | | | | |_) | (_) | | | (_|  __/
|___/\___|_|   \__,_|_| |_| |_| .__/ \___/|_|_|\___\___|
                              |_|
```

[![Build Status](https://travis-ci.org/pastjean/scrumpolice.svg?branch=master)](https://travis-ci.org/pastjean/scrumpolice)
[![Go Report Card](https://goreportcard.com/badge/github.com/pastjean/scrumpolice)](https://goreportcard.com/report/github.com/pastjean/scrumpolice)

Scrum bot ask every member of your team for a scrum report then reports it to
your team's channel at a predefined time.

- [Jason Fried's - Status meetings are the scourge](https://m.signalvnoise.com/status-meetings-are-the-scourge-39f49267ca90) started all the fuzz.

# Usage

The minimal configuration file has the following format:
```json
{
  "timezone": "America/Montreal",
  "teams": [
  ]
}
```
This will allow you to boot the scrumpolice without any teams, which can then be added directly by slack.

Run the bot with a slack bot user token:

```sh
SCRUMPOLICE_SLACK_TOKEN=xoxb-mytoken scrumpolice -databaseFile db.json
```

Command-line parameters:
* **databaseFile**: the path to a file that will be used to save the configuration of the scrumpolice. If the file is not found, the bot will attempt to load the data from file specified in the *config* parameter. Defaults to *db.json*.
* **config**: the path to a json configuration file that can be used to initialize the permanent config file. It will not be modified by any changes made by users of the bot. Defaults to *config.json*.

Full configuration file syntax:

```json
{
  "timezone": "America/Montreal",
  "teams": [
    {
      "channel": "themostaswesometeamchannel",
      "name": "L337 team",
      "members": [
        "@gfreeman",
        "@evance",
        "@wbreen"
      ],
      "split_report": true,
      "question_sets": [
        {
          "questions": [
            "What did you do yesterday?",
            "What will you do today?",
            "Are you being blocked by someone for a review? who ? why ?",
            "How will you dominate the world"
          ],
          "report_schedule_cron": "0 5 9 * * 1-5",
          "first_reminder_limit": "-50m",
          "last_reminder_limit": "-5m"
        }
      ]
    }
  ]
}
```

`split_report`: whether to post each scrum entry as a separate message or post all scrum entries in the same message.

# Development

Have a working go environment (since 1.8 just install go) otherwise you need the
`$GOPATH` set and use that instead of `$HOME/go`.

```sh
go get github.com/pastjean/scrumpolice
cd $HOME/go/src/github.com/pastjean/scrumpolice
dep ensure
# Run it
SCRUMPOLICE_SLACK_TOKEN=xoxb-mytoken go run cmd/scrumpolice/scrumpolice.go -config config.example.json
```
