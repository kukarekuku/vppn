package auth

import (
	"../shared/utils"
	"github.com/dropbox/godropbox/errors"
	"io/ioutil"
	"os"
	"strings"
)

var Key = ""

func Init() (err error) {
	pth := utils.GetAuthPath()

	if _, err := os.Stat(pth); os.IsNotExist(err) {
		Key, err = utils.RandStr(64)
		if err != nil {
			return
		}

		err = ioutil.WriteFile(pth, []byte(Key), os.FileMode(0644))
		if err != nil {
			err = errors.New("auth: Failed to auth key " + err.Error())
			return
		}
	} else {
		data, err := ioutil.ReadFile(pth)
		if err != nil {
			err = errors.New("auth: Failed to auth key " + err.Error())
			return
		}

		Key = strings.TrimSpace(string(data))

		if Key == "" {
			err = os.Remove(pth)
			if err != nil {
				err = errors.New("auth: Failed to reset auth key " + err.Error())
				return
			}
			Init()
		}
	}

	return
}
