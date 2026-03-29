package api

import "io"

// CloseQuietly closes the given io.Closer and suppresses the error.
// Use in defer statements to satisfy errcheck without noisy handling.
func CloseQuietly(c io.Closer) {
	if c != nil {
		_ = c.Close()
	}
}
