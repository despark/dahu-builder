package conf

import (
	"fmt"
	"github.com/spf13/viper"
)

type general struct {
	WorkDir      string
	Docker       Docker
	Terraform    Terraform
	Admin        Admin
	Repo         string
	Aws          Aws
	Environments []env
}

type env struct {
	Name   string
	Branch string
}

type Aws struct {
	Profile string
	Bucket  string
}

type Docker struct {
	Dir      string
	Image    string
	Username string
	Password string
}

type Admin struct {
	Repo   string
	Branch string
}

type Terraform struct {
	Dir string
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
