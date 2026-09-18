package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type errCollector struct {
	errs []error
}

func (c *errCollector) add(err error) {
	if err != nil {
		c.errs = append(c.errs, err)
	}
}

func (c *errCollector) err() error {
	if len(c.errs) == 0 {
		return nil
	}
	return errors.Join(c.errs...)
}

func (c *errCollector) duration(name, raw string, min time.Duration) time.Duration {
	d, err := time.ParseDuration(raw)
	if err != nil {
		c.add(fmt.Errorf("invalid %s: %w", name, err))
		return 0
	}
	if d < min {
		c.add(fmt.Errorf("%s must be at least %s, got %s", name, min, d))
	}
	return d
}

func (c *errCollector) intField(name, raw string) int {
	v, err := strconv.Atoi(raw)
	if err != nil {
		c.add(fmt.Errorf("invalid %s: %w", name, err))
	}
	return v
}

func (c *errCollector) required(name string) string {
	v, ok := os.LookupEnv(name)
	if !ok || v == "" {
		c.add(fmt.Errorf("%s is required", name))
	}
	return v
}

func getEnv(field, defaultValue string) string {
	value := os.Getenv(field)
	if value == "" {
		return defaultValue
	}
	return value
}
