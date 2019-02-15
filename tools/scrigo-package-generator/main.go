package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func error(err interface{}) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}

func goList(arg string) []string {
	cmd := exec.Command("go", "list", arg)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if arg == "stdlib" {
		cmd.Dir = "/usr/lib/golang/src"
		cmd.Args[2] = "./..."
	} else {
		error(fmt.Sprintf("unknown arg %s", arg))
	}
	err := cmd.Run()
	if err != nil {
		error(fmt.Sprintf("%v: %s", err, stderr.String()))
	}
	return strings.Split(stdout.String(), "\n")
}

func main() {
	switch len(os.Args) {
	case 2:
		var buf bytes.Buffer
		generatePackage(&buf, os.Args[1], "")
		fmt.Println(buf.String())
		return
	case 3:
		dir := os.Args[2]
		switch os.Args[1] {
		case "--stdlib":
			generateMultiplePackages(goList("stdlib"), dir)
		default:
			generateMultiplePackages(goList(os.Args[2]), dir)
		}
	default:
		error("bad usage")
	}
}
