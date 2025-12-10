package main

import (
	"flag"
	"io"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"text/template"
)

const dockerFileContents = `
  FROM debian:bookworm

  RUN apt-get update && apt-get install -y \
      crossbuild-essential-arm64 bc libssl-dev bison flex wget xz-utils

  COPY gokr-build-kernel /usr/bin/gokr-build-kernel

  RUN echo 'builduser:x:{{ .Uid }}:{{ .Gid }}:nobody:/:/bin/sh' >> /etc/passwd && \
      chown -R {{ .Uid }}:{{ .Gid }} /usr/src

  USER builduser
  WORKDIR /usr/src
  ENTRYPOINT /usr/bin/gokr-build-kernel
  `

var dockerFileTmpl = template.Must(template.New("dockerfile").Parse(dockerFileContents))

func copyFile(dest, src string) error {
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	_, err = io.Copy(out, in)
	return err
}

func getContainerExecutable() string {
	for _, exe := range []string{"podman", "docker"} {
		if _, err := exec.LookPath(exe); err == nil {
			return exe
		}
	}
	return "docker"
}

func main() {
	flag.Parse()
	executable := getContainerExecutable()

	tmp, err := os.MkdirTemp("/tmp", "gokr-rebuild-kernel")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	// Build the gokr-build-kernel binary for Linux
	cmd := exec.Command("go", "build", "-o", filepath.Join(tmp, "gokr-build-kernel"),
		"./cmd/gokr-build-kernel")
	cmd.Env = append(os.Environ(), "GOOS=linux", "CGO_ENABLED=0")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("go build: %v", err)
	}

	// Create Dockerfile
	u, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	dockerFile, err := os.Create(filepath.Join(tmp, "Dockerfile"))
	if err != nil {
		log.Fatal(err)
	}
	if err := dockerFileTmpl.Execute(dockerFile, struct{ Uid, Gid string }{u.Uid, u.Gid}); err != nil {
		log.Fatal(err)
	}
	dockerFile.Close()

	// Build container
	log.Printf("building %s container for kernel compilation", executable)
	build := exec.Command(executable, "build", "--rm=true", "--tag=gokr-rebuild-kernel", ".")
	build.Dir = tmp
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		log.Fatalf("%s build: %v", executable, err)
	}

	// Run container
	log.Printf("compiling kernel (this takes 10-30 minutes)")
	run := exec.Command(executable, "run", "--rm", "-v", tmp+":/tmp/buildresult:Z", "gokr-rebuild-kernel")
	run.Stdout = os.Stdout
	run.Stderr = os.Stderr
	if err := run.Run(); err != nil {
		log.Fatalf("%s run: %v", executable, err)
	}

	// Copy results back
	for _, file := range []string{"vmlinuz", "rk3528-nanopi-zero2.dtb"} {
		if err := copyFile(file, filepath.Join(tmp, file)); err != nil {
			log.Fatal(err)
		}
		log.Printf("wrote %s", file)
	}
}
