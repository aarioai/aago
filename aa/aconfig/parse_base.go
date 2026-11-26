package aconfig

import (
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// Extend adds other configs. Later entries replace keys that already exist.
func (c *Config) Extend(otherConfigs ...map[string]string) error {
	c.startWrite()
	defer c.endWrite()
	cfgs, err := c.parseOtherConfig(otherConfigs...)
	if err != nil || len(cfgs) == 0 {
		return err
	}

	// Keep the write-lock scope as small as possible.
	cfgMtx.Lock()
	for k, v := range cfgs {
		c.otherConfig[k] = v
	}
	cfgMtx.Unlock()
	return nil
}

func (c *Config) startWrite() {
	for {
		// Another goroutine may also be updating (concurrent config resets are almost nonexistent, but this handling is cheap, so do it this way).
		// Compare-and-swap the latest c.onWrite.Load() from false. If it is not false, another write is in progress and this one waits. If it is false, set it to true.
		if c.onWrite.CompareAndSwap(false, true) {
			break
		}
	}
}
func (c *Config) endWrite() {
	for {
		if c.onWrite.CompareAndSwap(true, false) {
			break
		}
	}
}

func (c *Config) initializeConfig() error {
	c.Env = NewEnv(c.GetWithFallback(ckEnv...))
	mock := c.GetWithFallback(ckMock...)
	if mock != "" && mock != "false" {
		c.Mock, _ = strconv.ParseBool(mock)
	}
	c.TextConfigDirs = textConfigDirs(c.GetWithFallback(ckTextConfigDirs...))
	return c.initializeTimezone()
}

func (c *Config) initializeTimezone() error {
	var err error
	tf := c.GetWithFallback(ckTimeFormat...)
	if tf == "" {
		tf = "2006-02-01 15:04:05"
	}
	c.TimeFormat = tf
	c.Timezone, _ = time.Now().Zone()
	c.TimeLocation = time.Local

	if tz := c.GetWithFallback(ckTimezone...); tz != "" {
		c.Timezone = tz
		if c.TimeLocation, err = time.LoadLocation(tz); err != nil {
			return fmt.Errorf("invalid timezone %s: %w", tz, err)
		}
		time.Local = c.TimeLocation
	}
	return nil
}

// Convert the space-separated config into a slice.
func (c *Config) loadAllTextConfigs(value string) (map[string]string, error) {
	dirs := textConfigDirs(value)
	if len(dirs) == 0 {
		return nil, nil
	}
	fileConfigs, err := c.loadTextConfigs(dirs[0])
	if len(dirs) == 1 || err != nil {
		return fileConfigs, err
	}
	if fileConfigs == nil {
		fileConfigs = make(map[string]string)
	}
	for i := 1; i < len(dirs); i++ {
		dir := dirs[i]
		cfg, err := c.loadTextConfigs(dir)
		if err != nil {
			return nil, err
		}
		maps.Copy(fileConfigs, cfg)
	}

	return fileConfigs, nil
}
func (c *Config) loadTextConfigs(dir string) (map[string]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read bin config directory %s: %w", dir, err)
	}

	rsaFiles := make(map[string]string, len(entries))
	// File configs come in pairs, so load the directory as a whole.
	for _, entry := range entries {
		if isNotValidFile(entry) {
			continue
		}
		if err = c.loadTextFile(dir, entry, rsaFiles); err != nil {
			return nil, err
		}
	}
	if len(rsaFiles) > 0 {
		return rsaFiles, nil
	}
	return nil, nil
}

// loadTextFile loads a single text config file.
func (c *Config) loadTextFile(root string, entry fs.DirEntry, rsaFiles map[string]string) error {
	filePath := filepath.Join(root, entry.Name())

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read bin config file %s: %w", filePath, err)
	}

	if len(data) == 0 {
		return fmt.Errorf("bin config file is empty: %s", filePath)
	}
	// directory name + "." + file name
	k := filepath.Base(root) + "." + entry.Name()
	v := string(data)
	if c.valueProcessor != nil {
		if v, err = c.valueProcessor(k, v); err != nil {
			return err
		}
	}
	rsaFiles[k] = v
	return nil
}

// Do not fetch at too fine a granularity, or errors become hard to trace.
func (c *Config) getTextData(name string) string {
	if c.isOnWrite() {
		cfgMtx.RLock()
		defer cfgMtx.RUnlock()
	}
	if len(c.textConfig) == 0 {
		return ""
	}
	return c.textConfig[name]
}
func (c *Config) isOnWrite() bool {
	return c.onWrite.Load()
}
