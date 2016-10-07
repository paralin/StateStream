package stream

import "time"

func DefaultStreamConfig() *Config {
	return &Config{
		RecordRate: &RateConfig{
			KeyframeFrequency: (time.Duration(60) * time.Second).Nanoseconds() / 1000000,
			ChangeFrequency:   (time.Duration(1) * time.Second).Nanoseconds() / 1000000,
		},
	}
}
