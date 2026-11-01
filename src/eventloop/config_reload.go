package eventloop

import (
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"

	"screen-ocr-llm/src/config"
	"screen-ocr-llm/src/llm"
	"screen-ocr-llm/src/logutil"
	"screen-ocr-llm/src/overlay"
)

type configSourceState struct {
	path        string
	digest      [sha256.Size]byte
	keyPath     string
	keyDigest   [sha256.Size]byte
	keyPresent  bool
	initialized bool
}

func (l *Loop) startConfigWatcher() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("event=config_watch_error error=%q", err)
		return
	}
	l.configWatcher = watcher
	l.watchedDirs = make(map[string]bool)
	l.watchConfigDirectory(config.SourcePath())
	if l.currentConfig != nil {
		l.watchConfigDirectory(l.currentConfig.APIKeyPath)
	}
}

// Watch the parent directory, so editors can replace the file atomically.
// The config file itself is never held open by the watcher.
func (l *Loop) watchConfigDirectory(path string) {
	if path == "" || l.configWatcher == nil {
		return
	}
	dir := filepath.Dir(path)
	if l.watchedDirs[dir] {
		return
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return
	}
	if err := l.configWatcher.Add(dir); err != nil {
		log.Printf("event=config_watch_error directory=%q error=%q", dir, err)
		return
	}
	l.watchedDirs[dir] = true
	log.Printf("event=config_watch_started directory=%q", dir)
}

func loadConfigSourceState(_ configSourceState, path string, keyPaths ...string) (configSourceState, error) {
	state := configSourceState{path: path, initialized: true}
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return configSourceState{}, fmt.Errorf("read config source %q: %w", path, err)
		}
		state.digest = sha256.Sum256(data)
	}
	if len(keyPaths) > 0 {
		state.keyPath = keyPaths[0]
		if data, err := os.ReadFile(state.keyPath); err == nil {
			state.keyDigest = sha256.Sum256(data)
			state.keyPresent = true
		}
	}
	return state, nil
}

func configSourceStateEqual(a, b configSourceState) bool {
	return a.initialized && b.initialized && a.path == b.path && a.digest == b.digest && a.keyPath == b.keyPath && a.keyDigest == b.keyDigest && a.keyPresent == b.keyPresent
}

func (l *Loop) checkRuntimeConfig() {
	if err := l.refreshRuntimeConfig(); err != nil {
		if message := err.Error(); message != l.lastReloadError {
			log.Printf("event=config_reload_failed source=%q error=%q retained_previous=true", config.SourcePath(), message)
			l.lastReloadError = message
		}
		return
	}
	if l.lastReloadError != "" {
		log.Printf("event=config_reload_recovered source=%q", config.SourcePath())
		l.lastReloadError = ""
	}
}

// refreshRuntimeConfig applies only complete, valid snapshots. The event loop
// invokes it while idle, and also before each capture request.
func (l *Loop) refreshRuntimeConfig() error {
	if !l.reloadConfigOnGrab {
		return nil
	}
	path := config.SourcePath()
	keyPath := ""
	if l.currentConfig != nil {
		keyPath = l.currentConfig.APIKeyPath
	}
	state, err := loadConfigSourceState(l.configState, path, keyPath)
	if err != nil {
		return err
	}
	if configSourceStateEqual(l.configState, state) {
		return nil
	}

	load := l.loadConfig
	if load == nil {
		load = config.LoadWithOptions
	}
	cfg, err := load(l.loadOptions)
	if err != nil {
		return fmt.Errorf("reload configuration: %w", err)
	}
	if cfg == nil {
		return fmt.Errorf("reload configuration: empty config")
	}
	if cfg.APIKey == "" {
		return fmt.Errorf("reload configuration: API key is missing (key file %q)", cfg.APIKeyPath)
	}
	if cfg.Model == "" {
		return fmt.Errorf("reload configuration: MODEL is missing")
	}
	if cfg.DefaultMode == "" {
		cfg.DefaultMode = config.DefaultModeRect
	}
	if cfg.OCRDeadlineSec <= 0 {
		cfg.OCRDeadlineSec = 20
	}
	state, err = loadConfigSourceState(state, path, cfg.APIKeyPath)
	if err != nil {
		return err
	}

	if l.currentConfig != nil && l.currentConfig.EnableFileLogging != cfg.EnableFileLogging {
		if err := logutil.Setup(cfg.EnableFileLogging); err != nil {
			return fmt.Errorf("set up file logging: %w", err)
		}
	}
	l.watchConfigDirectory(cfg.SourcePath)
	l.watchConfigDirectory(cfg.APIKeyPath)
	if l.selector == nil || l.defaultMode != cfg.DefaultMode {
		l.selector = overlay.NewSelector(cfg.DefaultMode)
	}
	initLLM := l.llmInit
	if initLLM == nil {
		initLLM = llm.Init
	}
	initLLM(&llm.Config{APIKey: cfg.APIKey, Model: cfg.Model, Providers: cfg.Providers})
	l.defaultMode = cfg.DefaultMode
	l.deadline = time.Duration(cfg.OCRDeadlineSec) * time.Second
	l.reloadConfigOnGrab = cfg.ReloadConfigOnGrab
	if l.currentConfig != nil && l.currentConfig.Hotkey != cfg.Hotkey {
		log.Printf("event=config_restart_required field=HOTKEY old=%q new=%q", l.currentConfig.Hotkey, cfg.Hotkey)
	}
	l.currentConfig = cfg
	l.configState = state
	log.Printf("event=config_reload_applied source=%q model=%q provider_order=%q api_key_present=true key_file=%q logging=%t default_mode=%q deadline_seconds=%d reload_on_grab=%t",
		path, cfg.Model, cfg.Providers, cfg.APIKeyPath, cfg.EnableFileLogging, cfg.DefaultMode, cfg.OCRDeadlineSec, cfg.ReloadConfigOnGrab)
	return nil
}
