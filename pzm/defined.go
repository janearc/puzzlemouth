package pzm

// The numbers here are true by definition.
const (
	// none is the count of nothing. It's defined.
	none = 0
	// one is a count of one, and the step from a length to the last
	// place. It's defined.
	one = 1
	// statusRan is the status of a command that worked. It's defined
	// by the shell.
	statusRan = 0
	// statusFailed is the status of a command that ran and failed. It's
	// defined by the shell.
	statusFailed = 1
	// statusUnread is the status of a command that could not be run, such
	// as for a flag given wrong. It's defined by the shell as misuse.
	statusUnread = 2
)
