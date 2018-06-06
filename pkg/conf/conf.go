package conf

import (
	"fmt"
	"github.com/spf13/viper"
)

type general struct {
	WorkDir      string
	DockerDir    string
	Repo         string
	Environments []env
}

type env struct {
	Name   string
	Branch string
}

type Conf struct {
	General general
}

func Init(path string, name string) Conf {
	viper.SetConfigName(name)
	viper.AddConfigPath(path)
	err := viper.ReadInConfig() // Find and read the config file

	if err != nil { // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	return readConfig()
}

func readConfig() Conf {
	c := Conf{general{}}

	err := viper.Unmarshal(&c.General)

	if err != nil {
		panic(err)
	}

	return c
}
