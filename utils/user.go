package utils

type User struct {
	Id uint

	// Permissions
	Group uint

	// Username
	Name string

	// Auth
	Email    string
	Password string

	// Confirm
	Confirmed bool
}
