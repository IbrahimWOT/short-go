package domain

import "errors"

var (
	// ErrURLNotFound is returned when the requested short URL does not exist.
	ErrURLNotFound = errors.New("url not found")

	// ErrURLExpired is returned when a short URL has passed its expiration time.
	ErrURLExpired = errors.New("url has expired")

	// ErrURLInactive is returned when a short URL has been deactivated.
	ErrURLInactive = errors.New("url is inactive")

	// ErrInvalidURL is returned when an input URL does not satisfy HTTP/HTTPS schema.
	ErrInvalidURL = errors.New("invalid url: must be an absolute http or https url")

	// ErrInvalidShortCode is returned when a custom or provided short code has invalid characters or length.
	ErrInvalidShortCode = errors.New("invalid short code: must be alphanumeric and between 3 and 16 characters")

	// ErrShortCodeConflict is returned when an attempt is made to use an already existing short code.
	ErrShortCodeConflict = errors.New("short code already in use")

	// ErrWorkerBufferFull is returned when the asynchronous analytics queue cannot accept more events.
	ErrWorkerBufferFull = errors.New("analytics worker queue buffer full")
)
