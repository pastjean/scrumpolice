package scrumpolice

type TeamState struct {
	name         string
	users        []string
	scrumReports map[string]*Report
}

func NewTeamState(name string, users []string, questions []string) *TeamState {
	scrumReports := map[string]*Report{}
	for _, user := range users {
		scrumReports[user] = NewReport(user, questions)
	}

	return &TeamState{name, users, scrumReports}
}
