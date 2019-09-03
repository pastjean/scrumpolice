package bot

import (
	"errors"
	"fmt"
	"regexp"
)

func getParams(regEx, url string) (paramsMap map[string]string) {

	var compRegEx = regexp.MustCompile(regEx)
	match := compRegEx.FindStringSubmatch(url)

	paramsMap = make(map[string]string)
	for i, name := range compRegEx.SubexpNames() {
		if i > 0 && i <= len(match) {
			paramsMap[name] = match[i]
		}
	}
	return
}

func getChannelFromMessage(receivedMessage string) (newChannel string, err error) {
	// https://regex101.com/r/FNZX6v/2/
	// When a user or public channel is sent, it gets converted:
	// @username -> <@U12345678>
	// #a_channel_name -> <#C12345678|a_channel_name>
	// When a private channel is sent, Slack does nothing with it
	params := getParams(`(?i)((<(@(?P<user_channel>[a-z0-9]+)|#[a-z0-9]+\|(?P<public_channel>.+))>)|#(?P<private_channel>[^\s]+)).*`, receivedMessage)
	fmt.Println(params)

	if len(params) == 0 || (params["user_channel"] == "" && params["public_channel"] == "" && params["private_channel"] == "") {
		err = errors.New("Cannot find channel name")
		return
	}

	newChannel = params["user_channel"]
	if params["public_channel"] != "" {
		newChannel = params["public_channel"]
	}
	if params["private_channel"] != "" {
		newChannel = params["private_channel"]
	}
	return
}