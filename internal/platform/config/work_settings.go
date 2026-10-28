package config

import (
	"log"
	"os"
	"strings"
	"time"
)

const abandonIdleGrace = 30 * time.Minute

const (
	defaultWorkIdleAfter    = 30 * time.Minute
	defaultWorkAbandonAfter = 2 * time.Hour
	defaultWorkSweepEvery   = time.Minute
)

func workDurations() (time.Duration, time.Duration, time.Duration) {
	idle := duration("WORK_IDLE_AFTER", defaultWorkIdleAfter)
	abandon := duration("WORK_ABANDON_AFTER", defaultWorkAbandonAfter)
	sweep := duration("WORK_SWEEP_EVERY", defaultWorkSweepEvery)
	minimum := idle + abandonIdleGrace
	if abandon < minimum {
		log.Printf("WORK_ABANDON_AFTER is below WORK_IDLE_AFTER plus 30m; using %s", minimum)
		abandon = minimum
	}
	if abandon < time.Hour || abandon > 3*time.Hour {
		log.Printf("WORK_ABANDON_AFTER=%s is outside the recommended 1h–3h range", abandon)
	}
	return idle, abandon, sweep
}

func workSweepMode() string {
	mode := strings.TrimSpace(os.Getenv("WORK_SWEEP_MODE"))
	if mode == "" {
		return "dry-run"
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
