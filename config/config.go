package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

var defaultFile = `$HOME/.gmap_notifier`

type folder struct {
	Name        string
	UnreadCount uint
	MsgCount    uint32
}

// Account stores the information required to read from a remote IMAP
// Server
type account struct {
	Host   string `json:"host"`
	Port   int    `json:"port,omitempty"`
	UseSSL bool   `json:"use_ssl"`

	User     string `json:"user"`
	Domain   string `json:"domain"`
	Password string `json:"password"`

	FolderList []string `json:"folders"`
	Folders    []folder
}

type Accounts []account

func (a *account) Server() string {
	return fmt.Sprintf("%v:%v", a.Host, a.Port)
}

func (a *account) UserName() string {
	return fmt.Sprintf("%s@%s", a.User, a.Domain)
}

func (a *account) setDefaults() {
	if a.Port == 0 {
		if a.UseSSL {
			a.Port = 993
		} else {
			a.Port = 143
		}
	}
	if a.Domain == "" {
		a.Domain = a.Host
	}
	for _, f := range a.FolderList {
		a.Folders = append(a.Folders, folder{
			Name: f,
		})
	}
}

type Config struct {
	Accounts   `json:"accounts"`
	ConfigFile string
}

func (a *Config) ReadConfig() error {
	if a.ConfigFile == "" {
		a.ConfigFile = defaultFile
	}

	a.ConfigFile = os.ExpandEnv(a.ConfigFile)

	return a.sourceConfigs()
}

func (a *Config) sourceConfigs() error {
	c, err := ioutil.ReadFile(a.ConfigFile)
	if err != nil {
		return err
	}

	var x map[string][]account

	if err := json.Unmarshal(c, &x); err != nil {
		return err
	}
	if x["accounts"] == nil {
		return fmt.Errorf("no accounts were found in %v", a.ConfigFile)
	}

	for _, l := range x["accounts"] {
		l.setDefaults()
		a.Accounts = append(a.Accounts, l)
	}

	return nil
}
