package utils

type User struct {
	Id uint

	// Permissions
	Group uint

	// Session hash
	Hash []byte

	// Username
	Name string

	// Auth
	Email    string
	Password string

	// Confirm
	Confirmed bool
}
