package app

import (
	"encoding/json"
	"flag"
	"os"
	"os/user"
	"path"
	"strings"

	"git.blackforestbytes.com/BlackForestBytes/goext/langext"
)

type Config struct {
	WebDAVURL  string `json:"webdav_url"`
	WebDAVUser string `json:"webdav_user"`
	WebDAVPass string `json:"webdav_pass"`

	LocalFallback string `json:"local_fallback"`

	WorkDir string `json:"work_dir"`

	ForceColors      bool   `json:"force_colors"`
	TerminalEmulator string `json:"terminal_emulator"`

	Debounce int `json:"debounce"`
}

func (app *Application) loadConfig() (Config, string) {
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

	var forceColors bool
	flag.BoolVar(&forceColors, "color", false, "Force color-output (default: auto-detect)")

	var terminalEmulator string
	flag.StringVar(&terminalEmulator, "terminal_emulator", "", "Command to start terminal-emulator, e.g. 'konsole -e'")

	var debounce int
	flag.IntVar(&debounce, "debounce", 0, "Debounce before sync (in seconds)")

	flag.Parse()

	if strings.HasPrefix(configPath, "~") {
		usr, err := user.Current()
		if err != nil {
			app.LogFatalErr("Failed to query users home directory", err)
		}

		configPath = strings.TrimPrefix(configPath, "~")
		configPath = path.Join(usr.HomeDir, configPath)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) && configPath != "" {

		te := ""
		if commandExists("konsole") {
			te = "konsole -e"
		} else if commandExists("gnome-terminal") {
			te = "gnome-terminal --"
		} else if commandExists("xterm") {
			te = "xterm -e"
		} else if commandExists("x-terminal-emulator") {
			te = "x-terminal-emulator -e"
		} else {
			app.LogError("Failed to determine terminal-emulator", nil)
		}

		_ = os.WriteFile(configPath, langext.Must(json.MarshalIndent(Config{
			WebDAVURL:        "https://your-nextcloud-domain.example/remote.php/dav/files/keepass.kdbx",
			WebDAVUser:       "",
			WebDAVPass:       "",
			LocalFallback:    "",
			WorkDir:          "/tmp/kpsync",
			Debounce:         3500,
			ForceColors:      false,
			TerminalEmulator: te,
		}, "", "    ")), 0644)
	}

	cfgBin, err := os.ReadFile(configPath)
	if err != nil {
		app.LogFatalErr("Failed to read config file from "+configPath, err)
	}

	var cfg Config
	err = json.Unmarshal(cfgBin, &cfg)
	if err != nil {
		app.LogFatalErr("Failed to parse config file from "+configPath, err)
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
	if forceColors {
		cfg.ForceColors = forceColors
	}
	if terminalEmulator != "" {
		cfg.TerminalEmulator = terminalEmulator
	}

	return cfg, configPath
}
