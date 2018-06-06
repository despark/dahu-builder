package builder

import (
	"bytes"
	"fmt"
	"github.com/otiai10/copy"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Build struct {
	Name       string
	Repo       string
	Branch     string
	WorkingDir string
	DockerDir  string
	Aws        Aws
	buildDir   string
}

var timestamp = strconv.FormatInt(time.Now().Unix(), 10)

func (b Build) Run(msg chan string) {
	b.buildDir = b.WorkingDir + "/" + b.Name + "/" + timestamp
	err := os.MkdirAll(b.buildDir, os.FileMode(0755))

	if err != nil {
		panic(err.Error())
	}

	b.checkout()

	msg <- b.Name
}

func (b Build) checkout() {
	gitBin := findCommand("git")
	srcDir := b.buildDir + "/src"
	_exec(gitBin, "clone", b.Repo, srcDir)
	_exec(gitBin, "-C", srcDir, "checkout", b.Branch)

	rev, _ := _exec(gitBin, "-C", srcDir, "rev-parse", "--short", "HEAD")

	version := rev + "-" + timestamp

	if os.RemoveAll(srcDir+"/.git") != nil {
		panic("Cannot remove .git folder")
	}

}

func _exec(name string, args ...string) (string, string) {
	var stdoutBuf, stderrBuf bytes.Buffer

	cmd := exec.Command(name, args...)
	fmt.Printf("Executing: %s\n\n", strings.Join(cmd.Args, " "))

	stdoutIn, _ := cmd.StdoutPipe()
	stderrIn, _ := cmd.StderrPipe()

	var errStdout, errStderr error
	stdout := io.MultiWriter(os.Stdout, &stdoutBuf)
	stderr := io.MultiWriter(os.Stderr, &stderrBuf)
	err := cmd.Start()
	if err != nil {
		log.Fatalf("cmd.Start() failed with '%s'\n", err)
	}

	go func() {
		_, errStdout = io.Copy(stdout, stdoutIn)
	}()

	go func() {
		_, errStderr = io.Copy(stderr, stderrIn)
	}()

	err = cmd.Wait()
	if err != nil {
		log.Fatalf("cmd.Run() failed with %s\n", err)
	}
	if errStdout != nil || errStderr != nil {
		log.Fatal("failed to capture stdout or stderr\n")
	}

	return string(stdoutBuf.Bytes()), string(stderrBuf.Bytes())
}

func findCommand(c string) string {

	s, e := exec.LookPath(c)
	if e != nil {
		panic(e.Error())
	}

	return s
}
