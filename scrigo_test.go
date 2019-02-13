package scrigo

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func executeScrigoSource(src []byte) string {
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	backupStdout := os.Stdout
	os.Stdout = w
	var b = &bytes.Buffer{}
	c := make(chan struct{})
	go func() {
		_, err = b.ReadFrom(r)
		if err != nil {
			panic(err)
		}
		c <- struct{}{}
	}()
	srcReader := bytes.NewReader(src)
	err = Run(srcReader)
	if err != nil {
		msg := err.Error()
		if msg[0] == ':' {
			return msg[1:]
		}
		return msg
	}
	err = os.Stdout.Close()
	if err != nil {
		panic(err)
	}
	_ = <-c
	os.Stdout = backupStdout
	return b.String()
}

// listUsedPackages lists packages used in src.
func listUsedPackages(src string) []string {
	possiblePkgs := []string{"fmt", "math", "time"}
	pkgs := []string{}
	for _, ppkg := range possiblePkgs {
		if strings.Contains(src, ppkg+".") {
			pkgs = append(pkgs, ppkg)
		}
	}
	return pkgs
}

var golangErrorReg = regexp.MustCompile(`[\w\. ]+:(\d+):(\d+):\s(.*)`)

func extractInfosFromGoErrorMessage(msg string) (int, int, string) {
	match := golangErrorReg.FindStringSubmatch(msg)
	line, _ := strconv.Atoi(match[1])
	column, _ := strconv.Atoi(match[2])
	errorMsg := match[3]
	errorMsg = strings.Replace(errorMsg, "syntax error: ", "", 1)
	return line, column, errorMsg
}

const goSkeleton string = `package main
func main() {
[scrigo]
}`

func executeGoSource(src string) string {
	tmpDir, err := ioutil.TempDir(os.TempDir(), "scrigotesting")
	if err != nil {
		panic(err)
	}
	defer func() {
		if !strings.HasPrefix(tmpDir, os.TempDir()) {
			panic(fmt.Errorf("invalid tmpDir: %q", tmpDir))
		}
		err := os.RemoveAll(tmpDir)
		if err != nil {
			panic(err)
		}
	}()
	tmpFile, err := os.Create(filepath.Join(tmpDir, "main.go"))
	if err != nil {
		panic(err)
	}
	pkgs := listUsedPackages(src)
	offset := 2 + len(pkgs)
	skel := goSkeleton
	for _, pkg := range pkgs {
		skel = strings.Replace(skel, "package main\n", "package main\nimport \""+pkg+"\"\n", 1)
	}
	goSrc := strings.Replace(skel, "[scrigo]", src, 1)
	_, err = tmpFile.WriteString(goSrc)
	if err != nil {
		panic(err)
	}
	cmd := exec.Command("go", "run", filepath.Base(tmpFile.Name()))
	cmd.Dir = tmpDir
	stdout := bytes.Buffer{}
	stderr := bytes.Buffer{}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	_ = cmd.Run()
	out := stdout.String() + stderr.String()
	if golangErrorReg.MatchString(out) {
		line, column, errorMsg := extractInfosFromGoErrorMessage(out)
		return fmt.Sprintf("%d:%d: %s", line-offset, column, errorMsg)
	}
	return out
}

func TestScrigo(t *testing.T) {
	t.Errorf("(skipping TestScrigo)")
	t.SkipNow()
	testDirs, err := ioutil.ReadDir("examples")
	if err != nil {
		t.Fatal(err)
	}
	availableTests := make([]string, 0, len(testDirs))
	for _, td := range testDirs {
		availableTests = append(availableTests, td.Name())
	}
	testSet := ""
	for _, arg := range os.Args {
		for _, test := range availableTests {
			if arg == test || arg == "all" {
				if testSet != "" {
					t.Fatalf("test type already set to %q,", testSet)
				}
				testSet = test
			}
		}
	}
	examples := []string{}
	if testSet == "" || testSet == "all" {
		for _, test := range availableTests {
			files, err := ioutil.ReadDir(filepath.Join("examples", test))
			if err != nil {
				t.Fatal(err)
			}
			for _, fi := range files {
				examples = append(examples, filepath.Join("examples", test, fi.Name()))
			}
		}
	} else {
		files, err := ioutil.ReadDir(filepath.Join("examples", testSet))
		if err != nil {
			t.Fatal(err)
		}
		for _, fi := range files {
			examples = append(examples, filepath.Join("examples", testSet, fi.Name()))
		}
	}
	for _, e := range examples {
		src, err := ioutil.ReadFile(e)
		if err != nil {
			t.Fatal(err)
		}
		scrigoOutput := executeScrigoSource(src)
		goOutput := executeGoSource(string(src))
		if scrigoOutput != goOutput {
			t.Errorf("%q:\nScrigo:     %q\nGo:         %q", e, scrigoOutput, goOutput)
		}
	}

}
