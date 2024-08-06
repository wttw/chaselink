package chaselink

import "time"

type Option func(*User) error

type ProgressCallback func(page Page) error

func UserAgent(userAgent string) Option {
	return func(u *User) error {
		u.UserAgent = userAgent
		return nil
	}
}

func Limit(limit int) Option {
	return func(u *User) error {
		u.Limit = limit
		return nil
	}
}

func Timeout(timeout time.Duration) Option {
	return func(u *User) error {
		u.Timeout = timeout
		return nil
	}
}

func Progress(callback ProgressCallback) Option {
	return func(u *User) error {
		u.Callback = callback
		return nil
	}
}
