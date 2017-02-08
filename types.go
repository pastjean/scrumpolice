package scrumpolice

type (
	TeamID string
	Team   struct {
		ID      TeamID
		Name    string
		Channel string
		Members []string
	}

	TeamService interface {
		Team(id TeamID) (*Team, error)
		CreateTeam(team *Team) error
		UpdateTeam(team *Team) error
		DeleteTeam(team *Team) error
	}
)
