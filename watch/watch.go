package watch

import (
	"../profile"
	"../shared/utils"
	"fmt"
	"github.com/op/go-logging"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"
)

var (
	lastRestart = time.Now()
	restartLock = sync.Mutex{}
	wake        = time.Now()
	wakeLock    = sync.Mutex{}
	log         = logging.MustGetLogger("watch")
)

func parseDns(data string) (searchDomains, searchAddresses []string) {
	dataSpl := strings.Split(data, "\n")
	key := ""
	searchDomains = []string{}
	searchAddresses = []string{}

	if len(dataSpl) < 2 {
		return
	}

	for _, line := range dataSpl[1 : len(dataSpl)-1] {
		if key == "" {
			if strings.Contains(line, "<array>") {
				key = strings.TrimSpace(strings.SplitN(line, ":", 2)[0])
				if key == "Pritunl" {
					key = ""
					continue
				}
			}
		} else {
			line = strings.TrimSpace(line)

			if strings.HasPrefix(line, "}") {
				key = ""
			} else {
				lineSpl := strings.SplitN(line, ":", 2)
				if len(lineSpl) > 1 {
					val := strings.TrimSpace(lineSpl[1])

					switch key {
					case "SearchDomains":
						searchDomains = append(searchDomains, val)
					case "ServerAddresses":
						if !strings.Contains(val, ":") {
							searchAddresses = append(searchAddresses, val)
						}
					}
				}
			}
		}
	}

	return
}

func wakeWatch(delay time.Duration) {
	defer func() {
		err := recover()
		if err != nil {
			log.Panic("watch: Panic", err)
			return
		}
	}()

	curTime := time.Now()
	delay += 1 * time.Second

	for {
		time.Sleep(delay)
		if time.Since(curTime) > 10*time.Second {
			reset := false

			wakeLock.Lock()
			if time.Since(wake) > 5*time.Second {
				wake = time.Now()
				reset = true
			}
			wakeLock.Unlock()

			if reset {
				restartLock.Lock()
				if time.Since(lastRestart) > 60*time.Second {
					lastRestart = time.Now()
					restartLock.Unlock()

					log.Warning("watch: Wakeup restarting...")

					profile.RestartProfiles(false)
				} else {
					restartLock.Unlock()
				}
			}
		}
		curTime = time.Now()
	}
}

func dnsWatch() {
	defer func() {
		err := recover()
		if err != nil {
			log.Panic("watch: Panic", err)
		}
	}()

	if runtime.GOOS != "darwin" {
		return
	}

	reset := false
	dnsState := false

	for {
		time.Sleep(1 * time.Second)

		if !profile.GetStatus() {
			if dnsState {
				err := utils.RestoreScutilDns()
				if err != nil {
					log.Warning("watch: Failed to restore DNS", err)
				} else {
					dnsState = false
				}
			}
			continue
		}

		vpn, _ := utils.GetScutilKey("State", "/Network/Pritunl/DNS")
		global, _ := utils.GetScutilKey("State", "/Network/Global/DNS")

		if strings.Contains(global, "No such key") {
			continue
		}

		dnsState = true

		if strings.Contains(vpn, "No such key") {
			connIds, err := utils.GetScutilConnIds()
			if err != nil {
				log.Error("watch: Failed to get DNS connection IDs", err)
				continue
			}

			if len(connIds) == 0 {
				continue
			}

			err = utils.CopyScutilKey(
				"State",
				fmt.Sprintf("/Network/Pritunl/Connection/%s", connIds[0]),
				"/Network/Pritunl/DNS",
			)
			if err != nil {
				log.Error("watch: Failed to copy DNS settings", err)
				continue
			}

			continue
		}

		vpnDomains, vpnAddresses := parseDns(vpn)
		globalDomains, globalAddresses := parseDns(global)

		if !reflect.DeepEqual(vpnDomains, globalDomains) ||
			!reflect.DeepEqual(vpnAddresses, globalAddresses) {

			if reset {
				restartLock.Lock()

				log.Warning("watch: Lost DNS settings updating...", vpnDomains, vpnAddresses, globalDomains, globalAddresses)

				err := utils.BackupScutilDns()
				if err != nil {
					log.Error("watch: Failed to backup DNS settings", err)
				} else {
					err = utils.CopyScutilDns("/Network/Pritunl/DNS")
					if err != nil {
						log.Error("watch: Failed to update DNS settings", err)
					}
				}

				restartLock.Unlock()
				reset = false
			} else {
				reset = true
			}
		} else {
			reset = false
		}
	}
}

func StartWatch() {
	go wakeWatch(10 * time.Millisecond)
	go wakeWatch(100 * time.Millisecond)
	go dnsWatch()
}
