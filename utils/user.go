package utils

type User struct {
	Id int

	// Permissions
	Group int

	// Username
	Name string

	// Auth
	Email    string
	Password string

	// Confirm
	Confirmed bool
}
