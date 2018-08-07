package main

import (
	"dahu-builder/builder"
	"dahu-builder/conf"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	cPath := flag.String("config", "deploy.json", "Build config file")
	flag.Var(&envs, "env", "Which environments to build")
	wDir := flag.String("workdir", "", "Override working directory")
	baseBuild := flag.Bool("base", false, "Create base build")
	noAdmin := flag.Bool("no-admin", false, "Don't make admin build")
	adminOnly := flag.Bool("admin-only", false, "Build only admin")
	flag.Parse()

	cDir := filepath.Dir(*cPath)
	cName := fileNoExt(filepath.Base(*cPath))

	c := conf.Init(cDir, cName)
	swDir := selectWorkDir(wDir)

	fmt.Println(c.General.WorkDir)

	ch := make(chan string)

	var envBuildLen = 0
	for _, v := range c.General.Environments {
		if envs[v.Name] {
			b := builder.Build{
				Env:        v.Name,
				Repo:       c.General.Repo,
				Docker:     c.General.Docker,
				Branch:     v.Branch,
				WorkingDir: swDir,
				Aws:        c.General.Aws,
				BaseBuild:  *baseBuild,
				Terraform:  c.General.Terraform,
				Admin:      c.General.Admin,
				NoAdmin:    *noAdmin,
				AdminOnly:  *adminOnly,
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

func fileNoExt(s string) string {
	n := strings.LastIndexByte(s, '.')
	if n >= 0 {
		return s[:n]
	}
	return s
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
