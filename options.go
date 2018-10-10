package monome

import "time"

type Option func(*connection)

func PollInterval(interval time.Duration) Option {
	return func(m *connection) {
		m.pollInterval = interval
	}
}
