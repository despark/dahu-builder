package main

import (
	"dahu-api-builder/pkg/builder"
	"dahu-api-builder/pkg/conf"
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"os"
	"path/filepath"
)

type mapStringFlags map[string]bool

func (i *mapStringFlags) String() string {
	return ""
}

func (i mapStringFlags) Set(value string) error {
	i[value] = true
	return nil
}

var envs = mapStringFlags{}

func main() {
	c := conf.Init("config", "config")

	flag.Var(&envs, "env", "Which environments to build")

	wdir := flag.String("workdir", "", "Override working directory")
	flag.Parse()

	swDir := selectWorkDir(wdir)

	fmt.Println(c.General.WorkDir)

	ch := make(chan string)

	awsSess := session.Must(session.NewSessionWithOptions(session.Options{
		Profile: "profile_name",
	}))

	aws := builder.Aws{
		Session: awsSess,
	}

	var envBuildLen = 0
	for _, v := range c.General.Environments {
		if envs[v.Name] {
			b := builder.Build{
				Name:       v.Name,
				Repo:       c.General.Repo,
				DockerDir:  c.General.DockerDir,
				Branch:     v.Branch,
				WorkingDir: swDir,
				Aws:        aws,
			}

			go b.Run(ch)
			envBuildLen++
		}
	}

	for i := 0; i < envBuildLen; i++ {
		fmt.Println()
		fmt.Printf("Environment %s is built!", <-ch)
	}
}

func selectWorkDir(wdir *string) string {
	if *wdir != "" {
		if _, err := os.Stat(*wdir); err == nil {
			return *wdir
		} else {
			panic(err.Error())
		}
	}

	return pwd()
}

func pwd() string {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex)
	return exPath
}
