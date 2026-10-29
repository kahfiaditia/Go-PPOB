package util

import (
	"os"

	"github.com/sirupsen/logrus"
)

var Log = logrus.New()

func InitLogger(env string) {
	Log.SetOutput(os.Stdout)
	if env == "production" {
		Log.SetFormatter(&logrus.JSONFormatter{})
		Log.SetLevel(logrus.InfoLevel)
	} else {
		Log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})
		Log.SetLevel(logrus.DebugLevel)
	}
}

// Helper agar log bisa di-trace by component
func WithField(key string, value interface{}) *logrus.Entry {
	return Log.WithField(key, value)
}
