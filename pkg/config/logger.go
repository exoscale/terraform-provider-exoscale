package config

import "log"

// LeveledTFLogger is a thin wrapper around stdlib.log that satisfies retryablehttp.LeveledLogger interface.
type LeveledTFLogger struct {
	Verbose bool
}

func (l LeveledTFLogger) Error(msg string, keysAndValues ...interface{}) {
	log.Println("[ERROR]", msg, keysAndValues)
}
func (l LeveledTFLogger) Info(msg string, keysAndValues ...interface{}) {
	log.Println("[INFO]", msg, keysAndValues)
}
func (l LeveledTFLogger) Debug(msg string, keysAndValues ...interface{}) {
	if l.Verbose {
		log.Println("[DEBUG]", msg, keysAndValues)
	}
}
func (l LeveledTFLogger) Warn(msg string, keysAndValues ...interface{}) {
	log.Println("[WARN]", msg, keysAndValues)
}
