package logger

import (
	"os"
	"sync"

	"github.com/sirupsen/logrus"
)

var (
	Log  *logrus.Logger
	once sync.Once
)

func Init() {
	once.Do(func() {
		Log = logrus.New()

		Log.SetLevel(logrus.DebugLevel)
		Log.SetOutput(os.Stdout)

		Log.SetFormatter(&logrus.TextFormatter{
			DisableColors: false,
			FullTimestamp: true,
		})
	})
}
