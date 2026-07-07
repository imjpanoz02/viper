package viper

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestWatchConfigConcurrentRead(t *testing.T) {
	// Create a temporary config file
	dir, err := ioutil.TempDir("", "viper_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	configFile := filepath.Join(dir, "config.yaml")

	writeConfig := func(val string) {
		content := []byte(fmt.Sprintf("key1: %s\nkey2: %s\n", val, val))
		if err := ioutil.WriteFile(configFile, content, 0666); err != nil {
			t.Fatal(err)
		}
	}

	writeConfig("value1")

	v := New()
	v.SetConfigFile(configFile)
	v.SetConfigType("yaml")
	if err := v.ReadInConfig(); err != nil {
		t.Fatal(err)
	}

	v.WatchConfig()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var wg sync.WaitGroup

	// Reader goroutines
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					k1 := v.GetString("key1")
					k2 := v.GetString("key2")
					if k1 != "" && k2 != "" && k1 != k2 {
						t.Errorf("Mismatch: key1=%q, key2=%q", k1, k2)
						return
					}
				}
			}
		}()
	}

	// Writer goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		val := "value2"
		for {
			select {
			case <-ctx.Done():
					return
			default:
				writeConfig(val)
				if val == "value1" {
					val = "value2"
				} else {
					val = "value1"
				}
				time.Sleep(50 * time.Millisecond)
			}
		}
	}()

	wg.Wait()
}
