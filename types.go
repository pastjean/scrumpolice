package scrumpolice

type Team struct {
	Members []*User
}

type User struct {
	Username string
}

type UserReport struct {
	Questions []string
	Answers   []string
}

// User can be part of multiple teams
// Teams can have multiple Members(Users)
