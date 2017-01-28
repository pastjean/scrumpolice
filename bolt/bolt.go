package bolt

import "github.com/scrumpolice/scrumpolice"

type UserService struct{}

// Ensure UserService implements scrumpolice.UserService
var _ scrumpolice.UserService = &UserService{}

func (s *UserService) User(id scrumpolice.UserID) (*scrumpolice.User, error) {
	panic("not implemented")
}
func (s *UserService) CreateUser(user *scrumpolice.User) error {
	panic("not implemented")
}
func (s *UserService) UpdateUser(user *scrumpolice.User) error {
	panic("not implemented")
}
func (s *UserService) DeleteUser(user *scrumpolice.User) error {
	panic("not implemented")
}

type TeamService struct{}

// Ensure TeamService implements scrumpolice.TeamService
var _ scrumpolice.TeamService = &TeamService{}

func (s *TeamService) Team(id scrumpolice.TeamID) (*scrumpolice.Team, error) {
	panic("not implemented")
}
func (s *TeamService) CreateTeam(team *scrumpolice.Team) error {
	panic("not implemented")
}
func (s *TeamService) UpdateTeam(team *scrumpolice.Team) error {
	panic("not implemented")
}
func (s *TeamService) DeleteTeam(team *scrumpolice.Team) error {
	panic("not implemented")
}
