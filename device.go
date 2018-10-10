package monome

type Device interface {
	// Rows returns the number of rows
	Rows() uint8

	// Cols returns the number of cols
	Cols() uint8

	// Set sets the button at position x,y to the given brightness
	// If the connection has been closed, nothing is sent
	// From the monome docs about brightness levels:
	// [0, 3] - off
	// [4, 7] - low intensity
	// [8, 11] - medium intensity
	// [12, 15] - high intensity
	// June 2012 devices allow the full 16 intensity levels.
	Set(x, y, brightness uint8) error

	// Switches the light at x,y on or off
	// Switches the light at x,y on or off
	// If on is true, it is a shortcut for  Set(x,y,15).
	// If on is false it is a shortcut for Set(x,y,0)
	Switch(x, y uint8, on bool) error

	// String returns an identifier as a string (name)
	String() string

	// ReadMessage reads a message from the device and calls the handler if necessary
	// It should normally not be called and is just there to allow external implementations of Device
	ReadMessage() error
}
