package internal_test

import (
	"reflect"
	"testing"

	"github.com/scrumpolice/scrumpolice"
	"github.com/scrumpolice/scrumpolice/bolt/internal"
)

func TestMarshalTeam(t *testing.T) {
	team := scrumpolice.Team{}

	var otherTeam scrumpolice.Team

	if buf, err := internal.MarshalTeam(&team); err != nil {
		t.Fatal(err)
	} else if err := internal.UnmarshalTeam(buf, &otherTeam); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(team, otherTeam) {
		t.Fatalf("unexpected copy: %#v", otherTeam)
	}
}

func TestMarshalUser(t *testing.T) {
	user := scrumpolice.User{}

	var otherUser scrumpolice.User

	if buf, err := internal.MarshalUser(&user); err != nil {
		t.Fatal(err)
	} else if err := internal.UnmarshalUser(buf, &otherUser); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(user, otherUser) {
		t.Fatalf("unexpected copy: %#v", otherUser)
	}
}
