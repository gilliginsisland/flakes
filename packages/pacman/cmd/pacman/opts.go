package main

import (
	"encoding/json"
	"os"

	"github.com/gilliginsisland/pacman/internal/app"
	"github.com/gilliginsisland/pacman/pkg/flagutil"
)

var opts Opts

type Opts struct {
	ConfigPath flagutil.Path     `short:"c" long:"config" description:"Path to the config file" default:"~/.config/pacman/config.json"`
	LogLevel   flagutil.LogLevel `short:"v" long:"verbosity" description:"Verbosity level"`
}

func (o *Opts) RuleSet() (*app.RuleSet, error) {
	f, err := os.Open(string(o.ConfigPath))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var rs app.RuleSet
	err = json.NewDecoder(f).Decode(&rs)
	if err != nil {
		return nil, err
	}
	rs.Path = string(o.ConfigPath)
	return &rs, nil
}
