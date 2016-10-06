package stream

import "time"

// Configuration for a stream
type Config struct {
	// Rate for incoming changes
	RecordRate RateConfig
}

// Configuration for rate
type RateConfig struct {
	// Minimum time between keyframes
	KeyframeFrequency time.Duration

	// Minimum time between changes.
	ChangeFrequency time.Duration
}

func DefaultStreamConfig() *Config {
	return &Config{
		RecordRate: RateConfig{
			KeyframeFrequency: time.Duration(60) * time.Second,
			ChangeFrequency:   time.Duration(1) * time.Second,
		},
	}
}
