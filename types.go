package scrumpolice

type TeamID string
type Team struct {
	ID TeamID
}

type TeamService interface {
	Team(id TeamID) (*Team, error)
	CreateTeam(team *Team) error
	UpdateTeam(team *Team) error
	DeleteTeam(team *Team) error
}

type UserID string
type User struct {
	ID UserID
}

type UserService interface {
	User(id UserID) (*User, error)
	CreateUser(user *User) error
	UpdateUser(user *User) error
	DeleteUser(user *User) error
}
