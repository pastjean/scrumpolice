package bot

import (
	"testing"
)


func TestGetChannelFromMessage(t *testing.T) {
	cases := []string{
		"<#C1234567|channelname>",
		"<#C1234567|channel_with_underscores>",
		"<#C1234567|café>",
		"<#C1234567|員>",
		"<@U1234567>",
		"#private_channel_name",
		"#private_channel_name with garbage",
	}
	casesExpected := []string{
		"channelname",
		"channel_with_underscores",
		"café",
		"員",
		"U1234567",
		"private_channel_name",
		"private_channel_name",
	}

	for i := 0; i < len(cases); i++ {
		channel, err := getChannelFromMessage(cases[i])
		if err != nil {
			t.Errorf("Error getting channel from %s", cases[i])
		}
		if channel != casesExpected[i] {
			t.Errorf("Error parsing channel %s, expected %s, got %s", cases[i], casesExpected[i], channel)
		}
	}

	casesFail := []string{"something_with_no_pound_or_at", "<C1234567>", "<C1234567|a_channel_name>"}

	for i := 0; i < len(casesFail); i++ {
		channel, err := getChannelFromMessage(casesFail[i])
		if err == nil {
			t.Errorf("Error: should have raised getting channel from %s, got %s", casesFail[i], channel)
		}
	}
}
