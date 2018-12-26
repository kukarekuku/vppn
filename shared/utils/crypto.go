package utils

import (
	"crypto/rand"
	"encoding/base64"
	"github.com/dropbox/godropbox/errors"
	"math"
	"math/big"
	mrand "math/rand"
	"regexp"
)

var (
	randRe = regexp.MustCompile("[^a-zA-Z0-9]+")
)

func RandStr(n int) (str string, err error) {
	for i := 0; i < 10; i++ {
		input, e := RandBytes(int(math.Ceil(float64(n) * 1.25)))
		if e != nil {
			err = e
			return
		}

		output := base64.RawStdEncoding.EncodeToString(input)
		output = randRe.ReplaceAllString(output, "")

		if len(output) < n {
			continue
		}

		str = output[:n]
		break
	}

	if str == "" {
		err = errors.New("utils: Random generate error " + err.Error())
	}

	return
}

func RandBytes(size int) (bytes []byte, err error) {
	bytes = make([]byte, size)
	_, err = rand.Read(bytes)
	if err != nil {
		err = errors.New("utils: Random read error " + err.Error())
	}
	return
}

func init() {
	n, err := rand.Int(rand.Reader, big.NewInt(9223372036854775806))
	if err != nil {
		log.Panic(err)
	}

	mrand.Seed(n.Int64())
}
