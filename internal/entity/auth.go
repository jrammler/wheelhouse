package entity

type User struct {
	Username     string
	PasswordHash string
	Roles        []string
}
