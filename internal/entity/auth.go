package entity

type User struct {
	Username     string   `json:"username"`
	PasswordHash string   `json:"password_hash"`
	Roles        []string `json:"roles"`
}
