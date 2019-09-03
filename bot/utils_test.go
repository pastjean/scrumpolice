package bot

import (
	"testing"
)


func TestGetChannelFromMessage(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		want        string
		wantFailure bool
	}{
		{"simple", "<#C1234567|channelname>", "channelname", false},
		{"With underscore", "<#C1234567|channel_with_underscores>", "channel_with_underscores", false},
		{"With accent", "<#C1234567|café>", "café", false},
		{"Japanese", "<#C1234567|員>", "員", false},
		{"User id", "<@U1234567>", "U1234567", false},
		{"Private","#private_channel_name", "private_channel_name", false},
		{"Private with spaces", "#private_channel_name with garbage", "private_channel_name", false},
		{"Fail no pound", "something_with_no_pound_or_at", "", true},
		{"Fail brackets no name", "<C1234567>", "", true},
		{"Fail brackets no pound", "<C1234567|a_channel_name>", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel, err := getChannelFromMessage(tt.input)
			if !tt.wantFailure {
				if err != nil {
					t.Errorf("Error getting channel from %s", tt.input)
				}
				if channel != tt.want {
					t.Errorf("Error parsing channel %s, expected %s, got %s", tt.input, tt.want, channel)
				}
			} else if err == nil {
				t.Errorf("Error: should have raised getting channel from %s, got %s", tt.input, channel)
			}
		})
	}
}
