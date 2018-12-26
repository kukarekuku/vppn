// Miscellaneous utils.
package utils

import (
	"../../config"
	"../command"
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/dropbox/godropbox/container/set"
	"github.com/dropbox/godropbox/errors"
	"github.com/op/go-logging"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	lockedInterfaces set.Set
	networkResetLock sync.Mutex
	log              = logging.MustGetLogger("utils")
)

func init() {
	lockedInterfaces = set.NewSet()
}

type Interface struct {
	Id   string
	Name string
}
type Interfaces []*Interface

func (intfs Interfaces) Len() int {
	return len(intfs)
}

func (intfs Interfaces) Swap(i, j int) {
	intfs[i], intfs[j] = intfs[j], intfs[i]
}

func (intfs Interfaces) Less(i, j int) bool {
	return intfs[i].Name < intfs[j].Name
}

func GetTaps() (interfaces []*Interface, err error) {
	interfaces = []*Interface{}

	cmd := command.Command("ipconfig", "/all")

	output, err := cmd.CombinedOutput()
	if err != nil {
		err = errors.New("utils: Failed to exec ipconfig " + err.Error())
		return
	}

	buf := bytes.NewBuffer(output)
	scan := bufio.NewReader(buf)

	intName := ""
	intTap := false
	intAddr := ""

	for {
		lineByte, _, e := scan.ReadLine()
		if e != nil {
			if e == io.EOF {
				break
			}
			err = e
			panic(err)
			return
		}
		line := string(lineByte)

		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "Ethernet adapter ") {
			intName = strings.Split(line, "Ethernet adapter ")[1]
			intName = intName[:len(intName)-1]
			intTap = false
			intAddr = ""
		} else if intName != "" {
			if strings.Contains(line, "TAP-Windows Adapter") {
				intTap = true
			} else if strings.Contains(line, "Physical Address") {
				intAddr = strings.Split(line, ":")[1]
				intAddr = strings.TrimSpace(intAddr)
			} else if intTap && intAddr != "" {
				intf := &Interface{
					Id:   intAddr,
					Name: intName,
				}
				interfaces = append(interfaces, intf)
				intName = ""
			}
		}
	}

	sort.Sort(Interfaces(interfaces))

	return
}

func AcquireTap() (intf *Interface, err error) {
	interfaces, err := GetTaps()
	if err != nil {
		return
	}

	for _, intrf := range interfaces {
		if !lockedInterfaces.Contains(intrf.Id) {
			lockedInterfaces.Add(intrf.Id)
			intf = intrf
			return
		}
	}

	return
}

func ReleaseTap(intf *Interface) {
	lockedInterfaces.Remove(intf.Id)
}

func GetScutilKey(typ, key string) (val string, err error) {
	cmd := command.Command("/usr/sbin/scutil")
	cmd.Stdin = strings.NewReader(
		fmt.Sprintf("open\nshow %s:%s\nquit\n", typ, key))

	output, err := cmd.CombinedOutput()
	if err != nil {
		err = errors.New("utils: Failed to exec scutil " + err.Error())
		return
	}

	val = strings.TrimSpace(string(output))

	return
}

func RemoveScutilKey(typ, key string) (err error) {
	cmd := command.Command("/usr/sbin/scutil")
	cmd.Stdin = strings.NewReader(
		fmt.Sprintf("open\nremove %s:%s\nquit\n", typ, key))

	err = cmd.Run()
	if err != nil {
		err = errors.New("utils: Failed to exec scutil " + err.Error())
		return
	}

	return
}

func CopyScutilKey(typ, src, dst string) (err error) {
	cmd := command.Command("/usr/sbin/scutil")
	cmd.Stdin = strings.NewReader(
		fmt.Sprintf("open\n"+
			"get %s:%s\n"+
			"set %s:%s\n"+
			"quit\n", typ, src, typ, dst))

	err = cmd.Run()
	if err != nil {
		err = errors.New("utils: Failed to exec scutil " + err.Error())
		return
	}

	return
}

func GetScutilService() (serviceId string, err error) {
	for i := 0; i < 20; i++ {
		data, e := GetScutilKey("State", "/Network/Global/IPv4")
		if e != nil {
			err = e
			return
		}

		dataSpl := strings.Split(data, "PrimaryService :")
		if len(dataSpl) < 2 {
			if i < 19 {
				time.Sleep(250 * time.Millisecond)
				continue
			}

			logrus.WithFields(logrus.Fields{
				"output": data,
			}).Error("utils: Failed to find primary service from scutil")

			err = errors.New("utils: Failed to find primary service from scutil")
			return
		}

		serviceId = strings.Split(dataSpl[1], "\n")[0]
		serviceId = strings.TrimSpace(serviceId)

		break
	}

	return
}

func RestoreScutilDns() (err error) {
	serviceId, err := GetScutilService()
	if err != nil {
		return
	}

	restoreKey := fmt.Sprintf("/Network/Pritunl/Restore/%s", serviceId)
	serviceKey := fmt.Sprintf("/Network/Service/%s/DNS", serviceId)

	data, err := GetScutilKey("State", restoreKey)
	if err != nil {
		return
	}

	if strings.Contains(data, "No such key") {
		return
	}

	data, err = GetScutilKey("State", serviceKey)
	if err != nil {
		return
	}

	if strings.Contains(data, "Pritunl : true") {
		err = CopyScutilKey("State", restoreKey, serviceKey)
		if err != nil {
			return
		}
	}

	data, err = GetScutilKey("Setup", serviceKey)
	if err != nil {
		return
	}

	if strings.Contains(data, "Pritunl : true") {
		data, err = GetScutilKey("Setup", restoreKey)
		if err != nil {
			return
		}

		if strings.Contains(data, "No such key") {
			err = RemoveScutilKey("Setup", serviceKey)
			if err != nil {
				return
			}
		} else {
			err = CopyScutilKey("Setup", restoreKey, serviceKey)
			if err != nil {
				return
			}
		}
	}

	ClearDNSCache()

	return
}

func CopyScutilDns(src string) (err error) {
	serviceId, err := GetScutilService()
	if err != nil {
		return
	}

	cmd := command.Command("/usr/sbin/scutil")
	cmd.Stdin = strings.NewReader(
		fmt.Sprintf("open\n"+
			"get State:%s\n"+
			"set State:/Network/Service/%s/DNS\n"+
			"set Setup:/Network/Service/%s/DNS\n"+
			"quit\n", src, serviceId, serviceId))

	err = cmd.Run()
	if err != nil {
		err = errors.New("utils: Failed to exec scutil " + err.Error())
		return
	}

	ClearDNSCache()

	return
}

func BackupScutilDns() (err error) {
	serviceId, err := GetScutilService()
	if err != nil {
		return
	}

	restoreKey := fmt.Sprintf("/Network/Pritunl/Restore/%s", serviceId)
	serviceKey := fmt.Sprintf("/Network/Service/%s/DNS", serviceId)

	data, err := GetScutilKey("State", serviceKey)
	if err != nil {
		return
	}

	if strings.Contains(data, "No such key") ||
		strings.Contains(data, "Pritunl : true") {

		return
	}

	err = CopyScutilKey("State", serviceKey, restoreKey)
	if err != nil {
		return
	}

	data, err = GetScutilKey("Setup", serviceKey)
	if err != nil {
		return
	}

	if strings.Contains(data, "No such key") {
		err = RemoveScutilKey("Setup", restoreKey)
		if err != nil {
			return
		}
	} else if !strings.Contains(data, "Pritunl : true") {
		err = CopyScutilKey("Setup", serviceKey, restoreKey)
		if err != nil {
			return
		}
	}

	return
}

func GetScutilConnIds() (ids []string, err error) {
	ids = []string{}

	cmd := command.Command("/usr/sbin/scutil")
	cmd.Stdin = strings.NewReader("open\nlist\nquit\n")

	output, err := cmd.CombinedOutput()
	if err != nil {
		err = errors.New("utils: Failed to exec scutil " + err.Error())
		return
	}

	for _, line := range strings.Split(string(output), "\n") {
		if !strings.Contains(line, "State:/Network/Pritunl/Connection/") {
			continue
		}

		spl := strings.Split(line, "State:/Network/Pritunl/Connection/")
		if len(spl) == 2 {
			ids = append(ids, strings.TrimSpace(spl[1]))
		}
	}

	return
}

func ClearScutilKeys() (err error) {
	remove := ""

	cmd := command.Command("/usr/sbin/scutil")
	cmd.Stdin = strings.NewReader("open\nlist\nquit\n")

	output, err := cmd.CombinedOutput()
	if err != nil {
		err = errors.New("utils: Failed to exec scutil " + err.Error())
		return
	}

	for _, line := range strings.Split(string(output), "\n") {
		if !strings.Contains(line, "State:/Network/Pritunl") {
			continue
		}

		if strings.Contains(line, "State:/Network/Pritunl/Restore") {
			continue
		}

		spl := strings.Split(line, "State:")
		if len(spl) != 2 {
			continue
		}

		key := strings.TrimSpace(spl[1])
		remove += fmt.Sprintf("remove State:%s\n", key)
	}

	if remove == "" {
		return
	}

	cmd = command.Command("/usr/sbin/scutil")
	cmd.Stdin = strings.NewReader("open\n" + remove + "quit\n")

	err = cmd.Run()
	if err != nil {
		err = errors.New("utils: Failed to exec scutil " + err.Error())
		return
	}

	return
}

func ResetNetworking() {
	logrus.Info("utils: Reseting networking")

	networkResetLock.Lock()
	defer networkResetLock.Unlock()

	switch runtime.GOOS {
	case "windows":
		command.Command("netsh", "interface", "ip", "delete",
			"destinationcache").Run()
		command.Command("ipconfig", "/release").Run()
		command.Command("ipconfig", "/renew").Run()
		command.Command("arp", "-d", "*").Run()
		command.Command("nbtstat", "-R").Run()
		command.Command("nbtstat", "-RR").Run()
		command.Command("ipconfig", "/flushdns").Run()
		command.Command("nbtstat", "/registerdns").Run()
		break
	case "darwin":
		cmd := command.Command("/usr/sbin/networksetup", "-getcurrentlocation")

		output, err := cmd.CombinedOutput()
		if err != nil {
			err = errors.New("utils: Failed to get network location " + err.Error())
			log.Error("utils: Reset networking error", err)
			return
		}

		location := strings.TrimSpace(string(output))

		if location == "pritunl-reset" {
			return
		}

		err = command.Command(
			"/usr/sbin/networksetup",
			"-createlocation",
			"pritunl-reset",
		).Run()

		if err != nil {
			err = errors.New("utils: Failed to create network location " + err.Error())
			log.Error("utils: Reset networking error", err)
		}

		command.Command("route", "-n", "flush").Run()

		err = command.Command(
			"/usr/sbin/networksetup",
			"-switchtolocation",
			"pritunl-reset",
		).Run()

		if err != nil {
			err = errors.New("utils: Failed to set network location " + err.Error())
			log.Error("utils: Reset networking error", err)
		}

		command.Command("route", "-n", "flush").Run()

		err = command.Command(
			"/usr/sbin/networksetup",
			"-switchtolocation",
			location,
		).Run()
		if err != nil {
			err = errors.New("utils: Failed to set network location " + err.Error())
			log.Error("utils: Reset networking error", err)
		}

		command.Command("route", "-n", "flush").Run()

		err = command.Command(
			"/usr/sbin/networksetup",
			"-deletelocation",
			"pritunl-reset",
		).Run()
		if err != nil {
			err = errors.New("utils: Failed to delete network location " + err.Error())
			log.Error("utils: Reset networking error", err)
		}
		break
	case "linux":
		output, _ := ExecOutput("/usr/bin/nmcli", "networking")
		if strings.Contains(output, "enabled") {
			command.Command("/usr/bin/nmcli", "connection", "reload").Run()
			command.Command("/usr/bin/nmcli", "networking", "off").Run()
			command.Command("/usr/bin/nmcli", "networking", "on").Run()
		}
		break
	default:
		log.Panic("profile: Not implemented")
	}
}

func ClearDNSCache() {
	switch runtime.GOOS {
	case "windows":
		command.Command("ipconfig", "/flushdns").Run()
		go func() {
			defer func() {
				err := recover()
				if err != nil {
					log.Panic("utils: Panic", err)
					return
				}
			}()

			for i := 0; i < 3; i++ {
				time.Sleep(1 * time.Second)
				command.Command("ipconfig", "/flushdns").Run()
			}
		}()
		break
	case "darwin":
		command.Command("killall", "-HUP", "mDNSResponder").Run()
		go func() {
			defer func() {
				err := recover()
				if err != nil {
					log.Panic("utils: Panic", err)
					return
				}
			}()

			for i := 0; i < 3; i++ {
				time.Sleep(1 * time.Second)
				command.Command("killall", "-HUP", "mDNSResponder").Run()
			}
		}()
		break
	case "linux":
		command.Command("systemd-resolve", "--flush-caches").Run()
		go func() {
			defer func() {
				err := recover()
				if err != nil {
					log.Panic("utils: Panic", err)
					return
				}
			}()

			for i := 0; i < 3; i++ {
				time.Sleep(1 * time.Second)
				command.Command("systemd-resolve", "--flush-caches").Run()
			}
		}()
		break
	default:
		log.Panic("profile: Not implemented")
	}
}

func Uuid() (id string) {
	idByte := make([]byte, 16)

	_, err := rand.Read(idByte)
	if err != nil {
		err = errors.New("utils: Failed to get random data " + err.Error())
		log.Panic(err)
	}

	id = hex.EncodeToString(idByte[:])

	return
}

func GetRootDir() (pth string) {
	pth, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Panic(err)
	}

	return
}

func GetAuthPath() (pth string) {
	if config.Development {
		pth = filepath.Join(GetRootDir(), "..", "dev")

		err := os.MkdirAll(pth, 0755)
		if err != nil {
			err = errors.New("utils: Failed to create dev directory" + err.Error())
			log.Panic(err)
		}

		pth = filepath.Join(pth, "auth")

		return
	}

	switch runtime.GOOS {
	case "windows":
		pth = filepath.Join("C:\\", "ProgramData", "Pritunl")

		err := os.MkdirAll(pth, 0755)
		if err != nil {
			err = errors.New("utils: Failed to create data directory " + err.Error())
			log.Panic(err)
		}

		pth = filepath.Join(pth, "auth")
		break
	case "darwin":
		pth = filepath.Join(string(os.PathSeparator), "Applications",
			"Pritunl.app", "Contents", "Resources", "auth")
		break
	case "linux":
		pth = filepath.Join(string(filepath.Separator),
			"var", "run", "pritunl.auth")
		break
	default:
		log.Panic("profile: Not implemented")
	}

	return
}

func GetLogPath() (pth string) {
	if config.Development {
		pth = filepath.Join(GetRootDir(), "..", "dev", "log")

		err := os.MkdirAll(pth, 0755)
		if err != nil {
			err = errors.New("utils: Failed to create dev directory " + err.Error())
			log.Panic(err)
		}

		pth = filepath.Join(pth, "pritunl.log")

		return
	}

	switch runtime.GOOS {
	case "windows":
		pth = filepath.Join("C:\\", "ProgramData", "Pritunl")

		err := os.MkdirAll(pth, 0755)
		if err != nil {
			err = errors.New("utils: Failed to create data directory " + err.Error())
			log.Panic(err)
		}

		pth = filepath.Join(pth, "pritunl.log")
		break
	case "darwin":
		pth = filepath.Join(string(os.PathSeparator), "Applications",
			"Pritunl.app", "Contents", "Resources", "pritunl.log")
		break
	case "linux":
		pth = filepath.Join(string(filepath.Separator),
			"var", "log", "pritunl.log")
		break
	default:
		log.Panic("profile: Not implemented")
	}

	return
}

func GetLogPath2() (pth string) {
	if config.Development {
		pth = filepath.Join(GetRootDir(), "..", "dev", "log")

		err := os.MkdirAll(pth, 0755)
		if err != nil {
			err = errors.New("utils: Failed to create dev directory " + err.Error())
			log.Panic(err)
		}

		pth = filepath.Join(pth, "pritunl.log.1")

		return
	}

	switch runtime.GOOS {
	case "windows":
		pth = filepath.Join("C:\\", "ProgramData", "Pritunl")

		err := os.MkdirAll(pth, 0755)
		if err != nil {
			err = errors.New("utils: Failed to create data directory " + err.Error())
			log.Panic(err)
		}

		pth = filepath.Join(pth, "pritunl.log.1")
		break
	case "darwin":
		pth = filepath.Join(string(os.PathSeparator), "Applications",
			"Pritunl.app", "Contents", "Resources", "pritunl.log.1")
		break
	case "linux":
		pth = filepath.Join(string(filepath.Separator),
			"var", "log", "pritunl.log.1")
		break
	default:
		log.Panic("profile: Not implemented")
	}

	return
}

func GetTempDir() (pth string, err error) {
	if config.Development {
		pth = filepath.Join(GetRootDir(), "..", "dev", "tmp")
		err = os.MkdirAll(pth, 0755)
		return
	}

	if runtime.GOOS == "windows" {
		pth = filepath.Join("C:\\", "ProgramData", "Pritunl")
		err = os.MkdirAll(pth, 0755)
	} else {
		pth = filepath.Join(string(filepath.Separator), "tmp", "pritunl")
		err = os.MkdirAll(pth, 0700)
	}

	if err != nil {
		err = errors.New("utils: Failed to create temp directory " + err.Error())
	}

	return
}

func GetPidPath() (pth string) {
	if config.Development {
		pth = filepath.Join(GetRootDir(), "..", "dev")

		err := os.MkdirAll(pth, 0755)
		if err != nil {
			err = errors.New("utils: Failed to create dev directory " + err.Error())
			log.Panic(err)
		}

		pth = filepath.Join(pth, "pritunl.pid")

		return
	}

	switch runtime.GOOS {
	case "darwin":
		pth = filepath.Join(string(os.PathSeparator), "Applications",
			"Pritunl.app", "Contents", "Resources", "pritunl.pid")
		break
	case "linux":
		pth = filepath.Join(string(filepath.Separator),
			"var", "run", "pritunl.pid")
		break
	default:
		log.Panic("profile: Not implemented")
	}

	return
}

func PidInit() (err error) {
	if runtime.GOOS == "windows" {
		return
	}

	pth := GetPidPath()
	pid := 0

	data, err := ioutil.ReadFile(pth)
	log.Error(err)
	if err != nil {
		if !os.IsNotExist(err) {
			err = errors.New(fmt.Sprintf("utils: Failed to read %s ", pth) + err.Error())
			return
		}
	} else {
		pidStr := strings.TrimSpace(string(data))
		if pidStr != "" {
			pid, _ = strconv.Atoi(pidStr)
		}
	}

	err = ioutil.WriteFile(
		pth,
		[]byte(strconv.Itoa(os.Getpid())),
		0644,
	)

	log.Error(err)
	if err != nil {
		err = errors.New(fmt.Sprintf("utils: Failed to write pid"))
		return
	}

	if pid != 0 {
		proc, e := os.FindProcess(pid)
		log.Error(e)
		if e == nil {
			// proc.Signal(os.Interrupt)

			done := false

			go func() {
				defer func() {
					recover()
				}()

				time.Sleep(5 * time.Second)

				if done {
					return
				}
				proc.Kill()
			}()

			proc.Wait()
			done = true

			time.Sleep(2 * time.Second)
		}
	}

	return
}