package proto

// A Backend provides Rooms and an implementation version.
type Backend interface {
	Close()

	// Gets a Room by name. If the Room doesn't already exist, it should
	// be created.
	GetRoom(name string) (Room, error)

	// Version returns the implementation version string.
	Version() string
}
