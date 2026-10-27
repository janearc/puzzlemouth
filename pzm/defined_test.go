package pzm_test

// The exit codes are defined by Tool.RunWith.
const (
	// exitOK is the code of a call that did what it was asked. It's
	// defined by Tool.RunWith.
	exitOK = 0
	// exitFailed is the code of a verb that ran and failed. It's defined
	// by Tool.RunWith.
	exitFailed = 1
	// exitRefused is the code of a tool that could not run. It's defined
	// by Tool.RunWith.
	exitRefused = 2
	// none is the count of nothing. It's defined.
	none = 0
)

// one is a count of one. It's defined.
const one = 1
