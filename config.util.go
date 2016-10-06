package stream

import "errors"

func (c *Config) Validate() error {
	def := DefaultStreamConfig()
	if c.RecordRate == nil {
		c.RecordRate = def.RecordRate
	}
	if err := c.RecordRate.Validate(); err != nil {
		return err
	}
	return nil
}

func (c *RateConfig) Validate() error {
	if c.ChangeFrequency < 0 || c.KeyframeFrequency == 0 {
		return errors.New("Frequency values must be >= 0.")
	}
	return nil
}
