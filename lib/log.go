package lib

import (
	"fmt"
	"log"
	"time"
)

func init() {
	log.SetFlags(0)
	log.SetOutput(new(logWriter))
}

type logWriter struct {
}

const logFormat = "2006-01-02 15:04:05.999"

func (writer logWriter) Write(bytes []byte) (int, error) {
	return fmt.Printf("[%s] %s", time.Now().UTC().Format(logFormat), string(bytes))
}
