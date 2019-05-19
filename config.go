package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
)

// Config holds server config
type Config struct {
	Path        string `json:"path,omitempty"` // config file path
	Port        int    `json:"port,omitempty"`
	Password    string `json:"password,omitempty"`     // one time, and it will be cleared after computed
	Iterations  int    `json:"iterations,omitempty"`   // safety, min 100,000
	Salt        string `json:"salt,omitempty"`         // salt for the crypt
	Crypt       string `json:"crypt,omitempty"`        // hashed password (password:salt)
	AllowedDirs string `json:"allowed_dirs,omitempty"` // directory allowed to be browse
}

// ConfigHashIterations how many times the password should be hashed
const ConfigHashIterations = 100000

// ConfigRead read and parse configuration file
func ConfigRead(filepath string) (*Config, error) {
	byteDat, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	var config Config

	err = json.Unmarshal(byteDat, &config)
	if err != nil {
		return nil, err
	}

	// sanity check
	if config.Port <= 0 || config.Port > 65535 {
		return nil, errors.New("invalid port number " + strconv.Itoa(config.Port))
	}
	if config.Crypt == "" && len(config.Password) < 6 {
		return nil, errors.New("password too short, min of 6")
	}

	// overwrite
	config.Path = filepath
	config.Iterations = ConfigHashIterations

	// hash password
	if config.Crypt == "" {
		// generate salt, longer because of limited character list
		config.Salt = generateSessionID(128)
		// calc password hash
		config.Crypt = SHA256Iter100k(config.Password, config.Salt)
		// clear password
		config.Password = ""
		// save new config file
		err := ConfigSave(&config, config.Path)
		if err != nil {
			fmt.Println("failed to save config file (b)")
			return nil, err
		}
	}

	return &config, nil
}

// ConfigSave save config to json file
func ConfigSave(config *Config, filepath string) error {
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
	err = ioutil.WriteFile(filepath, byteDat2, 0644)
	if err != nil {
		return err
	}

	return nil
}

// SHA256Iter100k iterate password and salt 100,000 times, return hex result
func SHA256Iter100k(password, salt string) string {
	str := password + ":" + salt
	bstr := []byte(str)

	var bstrx [sha256.Size]byte
	for i := 0; i < ConfigHashIterations; i++ {
		bstrx = sha256.Sum256(bstr)
		bstr = bstrx[0:32]
	}

	return fmt.Sprintf("%x", bstr)
}
