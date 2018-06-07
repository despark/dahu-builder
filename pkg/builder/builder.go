package builder

import (
	"bytes"
	"dahu-api-builder/pkg/conf"
	"fmt"
	"github.com/otiai10/copy"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Build struct {
	Env          string
	Repo         string
	Branch       string
	Docker       conf.Docker
	Terraform    conf.Terraform
	WorkingDir   string
	BaseBuild    bool
	Aws          conf.Aws
	buildDir     string
	srcDir       string
	version      string
	imageVersion string
	artifact     string
}

var timestamp = strconv.FormatInt(time.Now().Unix(), 10)

func (b Build) Run(msg chan string) {
	b.buildDir = b.WorkingDir + "/" + b.Env + "/" + timestamp
	b.srcDir = b.buildDir + "/src"
	err := os.MkdirAll(b.buildDir, os.FileMode(0755))

	if err != nil {
		panic(err.Error())
	}

	b.checkout()
	b.copy()
	b.ufo()
	b.makeArtifact()
	b.release()

	msg <- b.Env
}

func (b *Build) checkout() {
	gitBin := findCommand("git")
	_exec(gitBin, "", "clone", b.Repo, b.srcDir)
	_exec(gitBin, "", "-C", b.srcDir, "checkout", b.Branch)

	rev, _ := _exec(gitBin, "", "-C", b.srcDir, "rev-parse", "--short", "HEAD")

	b.version = rev + "-" + timestamp

	if os.RemoveAll(b.srcDir+"/.git") != nil {
		panic("Cannot remove .git folder")
	}

}

func (b Build) copy() {
	err := copy.Copy(b.Docker.Dir, b.buildDir)
	if err != nil {
		panic(err.Error())
	}
	awsBin := findCommand("aws")

	done := make(chan bool)

	go func(bin string, b Build, done chan bool) {
		_exec(awsBin,
			"",
			"--profile="+b.Aws.Profile,
			"s3", "sync",
			"s3://"+b.Aws.Bucket+"/var/"+b.Env+"/jwt",
			b.srcDir+"/var/"+b.Env+"/jwt")

		done <- true
	}(awsBin, b, done)

	go func(bin string, b Build, done chan bool) {
		_exec(awsBin,
			"",
			"--profile="+b.Aws.Profile,
			"s3", "sync",
			"s3://"+b.Aws.Bucket+"/var/"+b.Env+"/ios",
			b.srcDir+"/var/"+b.Env+"/ios")
		done <- true
	}(awsBin, b, done)

	go func(bin string, b Build, done chan bool) {
		_exec(awsBin,
			"",
			"--profile="+b.Aws.Profile,
			"s3", "cp",
			"s3://"+b.Aws.Bucket+"/config/parameters.yml."+b.Env,
			b.srcDir+"/app/config/parameters.yml")

		// If the file doesn't exist, create it, or append to the file
		f, err := os.OpenFile(b.srcDir+"/app/config/parameters.yml", os.O_APPEND|os.O_RDWR, 0644)
		if err != nil {
			panic(err.Error())
		}
		if _, err := f.Write([]byte("\n")); err != nil {
			panic(err.Error())
		}
		if err := f.Close(); err != nil {
			panic(err.Error())
		}

		done <- true
	}(awsBin, b, done)

	for i := 0; i < 3; i++ {
		<-done
	}

}

func (b Build) ufo() {
	ufoBin := findCommand("ufo")

	dockerLogin(b.Docker.Username, b.Docker.Password)

	_exec(ufoBin, b.buildDir, "init", "--app", "dahu-api", "--image", b.Docker.Image)

	if b.BaseBuild {
		_exec(ufoBin, b.buildDir, "docker", "base")
	}

	_exec(ufoBin, b.buildDir, "docker", "build", "--push")

	b.imageVersion, _ = _exec(ufoBin, b.buildDir, "docker", "name")
}

func (b *Build) makeArtifact() {
	dRun := b.buildDir + "/Dockerrun.aws.json"
	copy.Copy(b.buildDir+"/Dockerrun.aws.json.template", dRun)

	read, err := ioutil.ReadFile(dRun)
	errPanic(err)

	replace := strings.Replace(string(read), "%image%", b.imageVersion, 1)

	err = ioutil.WriteFile(dRun, []byte(replace), os.FileMode(0644))
	errPanic(err)

	b.artifact = b.buildDir + "/" + b.version + ".zip"
	a := NewArtifact(b.artifact)
	errPanic(err)

	ebExt := b.buildDir + "/.ebextensions/"
	files := map[string]string{
		dRun:  "/",
		ebExt: ".ebextensions/",
	}

	for file, archPath := range files {
		err = a.Add(file, archPath)
		errPanic(err)
	}
	err = a.Flush()
	errPanic(err)
}

func (b Build) release() {
	vt := b.buildDir + "/version.template"
	tid := b.Terraform.Dir + "/instances/" + b.Env
	vfp := tid + "/versions.tf"
	c, err := ioutil.ReadFile(vt)
	errPanic(err)

	newVersion := strings.Replace(string(c), "%version%", b.version, -1)

	f, err := os.OpenFile(vfp, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	errPanic(err)

	_, err = f.Write([]byte(newVersion))
	errPanic(err)

	err = f.Close()
	errPanic(err)

	artifactName := filepath.Base(b.artifact)

	awsBin := findCommand("aws")
	_exec(awsBin,
		"",
		"--profile="+b.Aws.Profile,
		"s3", "cp",
		b.artifact,
		"s3://"+b.Aws.Bucket+"/builds/"+artifactName)
}

func errPanic(err error) {
	if err != nil {
		panic(err.Error())
	}
}

func dockerLogin(u string, p string) {
	var stdoutBuf, stderrBuf bytes.Buffer

	dockerBin := findCommand("docker")

	buffer := bytes.NewReader([]byte(p))

	login := exec.Command(dockerBin, "login", "-u "+u, "--password-stdin")
	stdoutIn, _ := login.StdoutPipe()
	stderrIn, _ := login.StderrPipe()

	login.Stdin = buffer

	fmt.Printf("Executing: %s\n\n", strings.Join(login.Args, " "))

	var errStdout, errStderr error
	stdout := io.MultiWriter(os.Stdout, &stdoutBuf)
	stderr := io.MultiWriter(os.Stderr, &stderrBuf)

	err := login.Start()
	if err != nil {
		panic(fmt.Sprintf("cmd.Start() failed with '%s'\n", err))
	}

	go func() {
		_, errStdout = io.Copy(stdout, stdoutIn)
	}()

	go func() {
		_, errStderr = io.Copy(stderr, stderrIn)
	}()

	err = login.Wait()
	if err != nil {
		panic(fmt.Sprintf("cmd.Run() failed with %s\n", err))
	}

	if errStdout != nil || errStderr != nil {
		panic("failed to capture stdout or stderr\n")
	}
}

func _exec(name string, workdir string, args ...string) (string, string) {
	var stdoutBuf, stderrBuf bytes.Buffer

	cmd := exec.Command(name, args...)

	if workdir != "" {
		cmd.Dir = workdir
	}

	fmt.Printf("Executing: %s\n\n", strings.Join(cmd.Args, " "))

	stdoutIn, _ := cmd.StdoutPipe()
	stderrIn, _ := cmd.StderrPipe()

	var errStdout, errStderr error
	stdout := io.MultiWriter(os.Stdout, &stdoutBuf)
	stderr := io.MultiWriter(os.Stderr, &stderrBuf)
	err := cmd.Start()
	if err != nil {
		panic(fmt.Sprintf("cmd.Start() failed with '%s'\n", err))
	}

	go func() {
		_, errStdout = io.Copy(stdout, stdoutIn)
	}()

	go func() {
		_, errStderr = io.Copy(stderr, stderrIn)
	}()

	err = cmd.Wait()
	if err != nil {
		panic(fmt.Sprintf("cmd.Run() failed with %s\n", err))
	}
	if errStdout != nil || errStderr != nil {
		panic("failed to capture stdout or stderr\n")
	}

	return strings.TrimSuffix(string(stdoutBuf.Bytes()), "\n"), string(stderrBuf.Bytes())
}

func findCommand(c string) string {

	s, e := exec.LookPath(c)
	if e != nil {
		panic(e.Error())
	}

	return s
}
