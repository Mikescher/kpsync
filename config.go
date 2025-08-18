package kpsync

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/user"
	"strings"

	"git.blackforestbytes.com/BlackForestBytes/goext/langext"
	"mikescher.com/kpsync/log"
)

type Config struct {
	WebDAVURL  string `json:"webdav_url"`
	WebDAVUser string `json:"webdav_user"`
	WebDAVPass string `json:"webdav_pass"`

	LocalFallback string `json:"local_fallback"`

	WorkDir string `json:"work_dir"`

	Debounce int `json:"debounce"`
}

func LoadConfig() Config {
	var configPath string
	flag.StringVar(&configPath, "config", "~/.config/kpsync.json", "Path to the configuration file")

	var webdavURL string
	flag.StringVar(&webdavURL, "webdav_url", "", "WebDAV URL")

	var webdavUser string
	flag.StringVar(&webdavUser, "webdav_user", "", "WebDAV User")

	var webdavPass string
	flag.StringVar(&webdavPass, "webdav_pass", "", "WebDAV Password")

	var localFallback string
	flag.StringVar(&localFallback, "local_fallback", "", "Local fallback database")

	var workDir string
	flag.StringVar(&workDir, "work_dir", "", "Temporary working directory")

	var debounce int
	flag.IntVar(&debounce, "debounce", 0, "Debounce before sync (in seconds)")

	flag.Parse()

	if strings.HasSuffix(configPath, "~") {
		usr, err := user.Current()
		if err != nil {
			log.FatalErr("Failed to query users home directory", err)
		}
		fmt.Println(usr.HomeDir)

		configPath = strings.TrimSuffix(configPath, "~")
		configPath = fmt.Sprintf("%s/%s", usr.HomeDir, configPath)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) && configPath != "" {
		_ = os.WriteFile(configPath, langext.Must(json.Marshal(Config{
			WebDAVURL:     "https://your-nextcloud-domain.example/remote.php/dav/files/keepass.kdbx",
			WebDAVUser:    "",
			WebDAVPass:    "",
			LocalFallback: "",
			WorkDir:       "/tmp/kpsync",
			Debounce:      3500,
		})), 0644)
	}

	cfgBin, err := os.ReadFile(configPath)
	if err != nil {
		log.FatalErr("Failed to read config file from "+configPath, err)
	}

	var cfg Config
	err = json.Unmarshal(cfgBin, &cfg)
	if err != nil {
		log.FatalErr("Failed to parse config file from "+configPath, err)
	}

	if webdavURL != "" {
		cfg.WebDAVURL = webdavURL
	}
	if webdavUser != "" {
		cfg.WebDAVUser = webdavUser
	}
	if webdavPass != "" {
		cfg.WebDAVPass = webdavPass
	}
	if localFallback != "" {
		cfg.LocalFallback = localFallback
	}
	if workDir != "" {
		cfg.WorkDir = workDir
	}
	if debounce > 0 {
		cfg.Debounce = debounce
	}

	return cfg
}
