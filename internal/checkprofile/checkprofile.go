package checkprofile

import (
	"errors"
	"log"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config object
type Config struct {
	Accounts []Account
	System   System
}

// System for basic auth
type System struct {
	Account    string
	Password   string
	ReNewTiming uint64
	DdnsTiming uint64
}

// Account struct
type Account struct {
	Username string
	Password string
	ZoneName   string
	RecordName string
}

// ReadConf will decode data
func ReadConf(filename string) (*Config, error) {
	var (
		conf *Config
		err  error
	)
	filename, err = filepath.Abs(filename)
	if err != nil {
		log.Fatal(err)
		return conf, err
	}
	if _, err = toml.DecodeFile(filename, &conf); err != nil {
		log.Fatal(err)
	}

	if conf.System.DdnsTiming < 5 || conf.System.ReNewTiming < 5 {
		log.Fatal(errors.New("err"))
	}
	return conf, err
}
