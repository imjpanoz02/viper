package viper

import (
	"log"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

// OnConfigChange sets the event handler that is called when the config file changes.
func (v *Viper) OnConfigChange(run func(in fsnotify.Event)) {
	v.onConfigChange = run
}

// WatchConfig starts watching the config file for changes.
func (v *Viper) WatchConfig() {
	initWatcher()

	configFile := filepath.Clean(v.configFile)
	configDir, _ := filepath.Split(configFile)
	realConfigFile, _ := filepath.EvalSymlinks(configFile)

	watcher.Add(configDir)

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				currentConfigFile, _ := filepath.EvalSymlinks(configFile)
				// we only care about the config file with the absolute path
				if filepath.Clean(event.Name) == configFile || filepath.Clean(event.Name) == realConfigFile || filepath.Clean(event.Name) == currentConfigFile {
					if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
						// Create a temporary Viper instance to read the config
						v2 := New()
						v.mu.RLock()
						v2.configFile = v.configFile
						v2.configType = v.configType
						v2.configName = v.configName
						v2.configPaths = v.configPaths
						v2.keyDelim = v.keyDelim
						v.mu.RUnlock()

						err := v2.ReadInConfig()
						if err != nil {
							log.Println("error:", err)
							continue
						}

						v.mu.Lock()
						v.config = v2.config
						if v.configFile == "" {
							v.configFile = v2.configFile
						}
						v.mu.Unlock()

						if v.onConfigChange != nil {
							v.onConfigChange(event)
						}
					}
				}
			}
		}
	}()
}

var watcher *fsnotify.Watcher

func initWatcher() {
	if watcher != nil {
		return
	}

	var err error
	watcher, err = fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
}
