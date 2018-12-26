package auth

import (
	"../shared/utils"
	"github.com/op/go-logging"
	"io/ioutil"
	"os"
	"strings"
)

var (
	log = logging.MustGetLogger("auth")
)

var Key = ""

func Init() {
	pth := utils.GetAuthPath()

	if _, err := os.Stat(pth); os.IsNotExist(err) {
		Key, err = utils.RandStr(64)
		if err != nil {
			log.Error(err)
			return
		}

		err = ioutil.WriteFile(pth, []byte(Key), os.FileMode(0644))
		if err != nil {
			log.Error("auth: Failed to auth key", err)
			return
		}
	} else {
		data, err := ioutil.ReadFile(pth)
		if err != nil {
			log.Error("auth: Failed to auth key ", err)
			return
		}

		Key = strings.TrimSpace(string(data))

		if Key == "" {
			err = os.Remove(pth)
			if err != nil {
				log.Error("auth: Failed to reset auth key", err)
				return
			}
			Init()
		}
	}
}
