package main

import (
	"github.com/op/go-logging"
)

var (
	log = logging.MustGetLogger("main")
)

func main() {
	log.Info("Start service")

	log.Info("All done")
}
