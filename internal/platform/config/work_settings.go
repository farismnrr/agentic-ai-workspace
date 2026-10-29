package config

import (
	"log"
	"os"
	"strings"
	"time"
)

const (
	defaultWorkIdleAfter    = 10 * time.Minute
	defaultWorkAbandonAfter = 15 * time.Minute
	defaultWorkSweepEvery   = time.Minute
)

func workDurations() (time.Duration, time.Duration, time.Duration) {
	idle := duration("WORK_IDLE_AFTER", defaultWorkIdleAfter)
	abandon := duration("WORK_ABANDON_AFTER", defaultWorkAbandonAfter)
	sweep := duration("WORK_SWEEP_EVERY", defaultWorkSweepEvery)
	if sweep < time.Second {
		log.Printf("WORK_SWEEP_EVERY must be at least 1s; using 1s")
		sweep = time.Second
	}
	if abandon <= idle {
		minimum := idle + time.Second
		log.Printf("WORK_ABANDON_AFTER must be greater than WORK_IDLE_AFTER; using %s", minimum)
		abandon = minimum
	}

	return idle, abandon, sweep
}

func workSweepMode() string {
	mode := strings.TrimSpace(os.Getenv("WORK_SWEEP_MODE"))
	if mode == "" {
		return "on"
	}
	if mode != "on" && mode != "dry-run" && mode != "off" {
		log.Printf("WORK_SWEEP_MODE must be on, dry-run, or off; using dry-run")
		return "dry-run"
	}
	return mode
}

func duration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		log.Printf("%s must be a positive duration; using %s", key, fallback)
		return fallback
	}
	return parsed
}
