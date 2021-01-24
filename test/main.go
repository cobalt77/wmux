package main

import (
	"os"
	"time"

	"github.com/cobalt77/wmux"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetLevel(logrus.InfoLevel)
	f, _ := os.Create("./log.txt")
	logrus.SetOutput(f)

	m, _ := wmux.NewMultiplexer(os.Stdout, wmux.Dimensions{
		Rows:           3,
		Columns:        3,
		MinColumnWidth: 20,
		Height:         5,
	})

	done := make(chan struct{}, 1)

	go func() {
		writers := m.Writers()
		for {
			for _, w := range writers {
				time.Sleep(16 * time.Millisecond)
				w.Write([]byte(time.Now().String()))
			}
		}
	}()
	m.Run(done)
}
