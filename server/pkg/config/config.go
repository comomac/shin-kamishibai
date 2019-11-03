package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/comomac/shin-kamishibai/server/pkg/lib"
)

// Config holds server config
type Config struct {
	Path         string   `json:"path,omitempty"`     // config file path
	Port         int      `json:"port"`               // server port
	DBPath       string   `json:"db_path"`            // where db file is stored
	Username     string   `json:"username"`           // username for the http authentication
	Password     string   `json:"password,omitempty"` // one time, and it will be cleared after computed
	Iterations   int      `json:"iterations"`         // safety, min 100,000
	Salt         string   `json:"salt"`               // salt for the crypt
	Crypt        string   `json:"crypt"`              // password hash
	AllowedDirs  []string `json:"allowed_dirs"`       // directory allowed to be browse
	ImageResize  bool     `json:"image_resize"`       // resize images in reader
	ImageQuality int      `json:"image_quality"`      // image quality for resized image
}

// ConfigHashIterations how many times the password should be hashed
const ConfigHashIterations = 100000

// Read read and parse configuration file
func Read(fpath string) (*Config, error) {
	byteDat, err := ioutil.ReadFile(fpath)
	if err != nil {
		return nil, err
	}

	var cfg Config

	err = json.Unmarshal(byteDat, &cfg)
	if err != nil {
		return nil, err
	}

	// sanity check
	if cfg.Port <= 0 || cfg.Port > 65535 {
		return nil, errors.New("invalid port number " + strconv.Itoa(cfg.Port))
	}
	if cfg.Crypt == "" && len(cfg.Password) < 6 {
		return nil, errors.New("password too short, min of 6")
	}
	if len(cfg.Username) < 3 {
		return nil, errors.New("username too short, min length 3")
	}

	// overwrite
	cfg.Path = fpath
	cfg.Iterations = ConfigHashIterations

	if cfg.DBPath == "" {
		cfg.DBPath = filepath.Dir(fpath) + "/db.txt"
	}

	// hash password
	if cfg.Crypt == "" {
		// generate salt, longer because of limited character list
		cfg.Salt = lib.GenerateString(128)
		// calc password hash
		cfg.Crypt = lib.SHA256Iter(cfg.Password, cfg.Salt, ConfigHashIterations)
		// clear password
		cfg.Password = ""
		// save new cfg file
		err := Save(&cfg, cfg.Path)
		if err != nil {
			fmt.Println("failed to save config file (b)")
			return nil, err
		}
	}

	// create thumbnail cache dir if not exists
	cacheDir := filepath.Join(filepath.Dir(cfg.Path), "cache")
	err = os.MkdirAll(cacheDir, os.ModePerm)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save save config to json file
func Save(config *Config, fpath string) error {
	// create a copy
	config2 := config
	// clear, setup setting
	config2.Path = ""
	config2.Password = ""
	config2.Iterations = ConfigHashIterations

	// save to file
	byteDat2, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(fpath, byteDat2, 0644)
	if err != nil {
		return err
	}

	return nil
}
