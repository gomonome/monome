package monome

import "time"

type Option func(*monome)

func PollInterval(interval time.Duration) Option {
	return func(m *monome) {
		m.pollInterval = interval
	}
}
