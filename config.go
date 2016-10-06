package stream

import "time"

func DefaultStreamConfig() *Config {
	return &Config{
		RecordRate: &RateConfig{
			KeyframeFrequency: float64(time.Duration(60) * time.Second),
			ChangeFrequency:   float64(time.Duration(1) * time.Second),
		},
	}
}
