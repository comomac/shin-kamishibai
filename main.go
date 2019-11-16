package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/comomac/shin-kamishibai/pkg/config"
	"github.com/comomac/shin-kamishibai/pkg/fdb"
	"github.com/comomac/shin-kamishibai/pkg/lib"
	svr "github.com/comomac/shin-kamishibai/server"
)

func loadDirs(db *fdb.FlatDB, allowedDirs []string) {
	for _, dir := range allowedDirs {
		err := fdb.AddDirR(db, dir)
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
		cfgFilePath = filepath.Join(lib.UserHome(), cfgFilePath[2:])
	}

	cfg, err := config.Read(cfgFilePath)
	if err != nil {
		fmt.Println("failed to read config file")
		panic(err)
	}

	// new db
	db := fdb.New(cfg.DBPath)
	db.Load()
	// load all books recursively
	go loadDirs(db, cfg.AllowedDirs)

	svr.Start(cfg, db)
}
