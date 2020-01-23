package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"
)

func loadDirs(db *FlatDB, allowedDirs []string) {
	for _, dir := range allowedDirs {
		err := db.AddDirR(dir)
		if err != nil {
			fmt.Println("failed to add dir -", err)
		}
	}
	fmt.Println("dirs loaded")
}

func main() {
	// use config on local dir by default if no param given
	xConfDir := flag.String("conf-dir", "~/etc/shin-kamishibai/config.json", "full path of the configuration file")
	flag.Parse()

	cfgFilePath := *xConfDir

	// use home path if ~/ exists
	if strings.HasPrefix(cfgFilePath, "~/") {
		cfgFilePath = filepath.Join(UserHome(), cfgFilePath[2:])
	}

	config := &Config{}
	err := config.Read(cfgFilePath)
	if err != nil {
		fmt.Println("failed to read config file")
		panic(err)
	}

	fmt.Printf("%+v", config)

	// new db
	db := &FlatDB{}
	db.New(config.PathDB)
	db.Load()
	// load all books recursively
	go loadDirs(db, config.AllowedDirs)

	svr := Server{
		Database: db,
		Config:   config,
	}
	svr.Start()
}
