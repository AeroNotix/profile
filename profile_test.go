package profile

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type checkFn func(t *testing.T, stdout, stderr []byte, err error)

var profileTests = []struct {
	name   string
	code   string
	checks []checkFn
}{{
	name: "default profile (cpu)",
	code: `
package main

import "github.com/pkg/profile"

func main() {
	defer profile.Start().Stop()
}	
`,
	checks: []checkFn{
		NoStdout,
		Stderr("profile: cpu profiling enabled"),
		NoErr,
	},
}, {
	name: "memory profile",
	code: `
package main

import "github.com/pkg/profile"

func main() {
	defer profile.Start(profile.MemProfile).Stop()
}	
`,
	checks: []checkFn{
		NoStdout,
		Stderr("profile: memory profiling enabled"),
		NoErr,
	},
}, {
	name: "memory profile (rate 2048)",
	code: `
package main

import "github.com/pkg/profile"

func main() {
	defer profile.Start(profile.MemProfileRate(2048)).Stop()
}	
`,
	checks: []checkFn{
		NoStdout,
		Stderr("profile: memory profiling enabled (rate 2048)"),
		NoErr,
	},
}, {
	name: "double start",
	code: `
package main

import "github.com/pkg/profile"

func main() {
	defer profile.Start().Stop()
	defer profile.Start().Stop()
}	
`,
	checks: []checkFn{
		NoStdout,
		Stderr("cpu profiling enabled", "profile: Start() already called"),
		Err,
	},
}, {
	name: "block profile",
	code: `
package main

import "github.com/pkg/profile"

func main() {
	defer profile.Start(profile.BlockProfile).Stop()
}	
`,
	checks: []checkFn{
		NoStdout,
		Stderr("profile: block profiling enabled"),
		NoErr,
	},
}, {
	name: "profile path",
	code: `
package main

import "github.com/pkg/profile"

func main() {
	defer profile.Start(profile.ProfilePath(".")).Stop()
}	
`,
	checks: []checkFn{
		NoStdout,
		Stderr("profile: cpu profiling enabled, cpu.pprof"),
		NoErr,
	},
}, {
	name: "profile path error",
	code: `
package main

import "github.com/pkg/profile"

func main() {
		defer profile.Start(profile.ProfilePath("README.md")).Stop()
}	
`,
	checks: []checkFn{
		NoStdout,
		Stderr("could not create initial output"),
		Err,
	},
}, {
	name: "profile quiet",
	code: `
package main

import "github.com/pkg/profile"

func main() {
        defer profile.Start(profile.Quiet).Stop()
}       
`,
	checks: []checkFn{NoStdout, NoStderr, NoErr},
}}

func TestProfile(t *testing.T) {
	for _, tt := range profileTests {
		t.Log(tt.name)
		stdout, stderr, err := runTest(t, tt.code)
		for _, f := range tt.checks {
			f(t, stdout, stderr, err)
		}
	}
}

// Stdout verifies that the given lines match the output from stdout
func Stdout(lines ...string) checkFn {
	return func(t *testing.T, stdout, _ []byte, _ error) {
		buf := bytes.NewBuffer(stdout)
		if !validateOutput(buf, lines) {
			t.Errorf("stderr: wanted '%s', got '%s'", lines, stdout)
		}

	}
}

// NoStdout checks that stdout was blank.
func NoStdout(t *testing.T, stdout, _ []byte, _ error) {
	if len := len(stdout); len > 0 {
		t.Errorf("stdout: wanted 0 bytes, got %d", len)
	}
}

// Stderr verifies that the given lines match the output from stderr
func Stderr(lines ...string) checkFn {
	return func(t *testing.T, stdout, stderr []byte, _ error) {
		buf := bytes.NewBuffer(stderr)
		if !validateOutput(buf, lines) {
			t.Errorf("stderr: wanted '%s', got '%s'", lines, stderr)
		}
	}
}

// NoStderr checks that stderr was blank.
func NoStderr(t *testing.T, _, stderr []byte, _ error) {
	if len := len(stderr); len > 0 {
		t.Errorf("stderr: wanted 0 bytes, got %d", len)
	}
}

// Err checks that there was an error returned
func Err(t *testing.T, _, _ []byte, err error) {
	if err == nil {
		t.Errorf("expected error")
	}
}

// NoErr checks that err was nil
func NoErr(t *testing.T, _, _ []byte, err error) {
	if err != nil {
		t.Errorf("error: expected nil, got %v", err)
	}
}

func validateOutput(buf *bytes.Buffer, lines []string) bool {
	s := bufio.NewScanner(buf)
	for _, l := range lines {
		if !s.Scan() {
			return false
		}
		if !strings.Contains(s.Text(), l) {
			return false
		}
	}
	return true
}

// runTest executes the go program supplied and returns the contents of stdout,
// stderr, and an error which may contain status information about the result
// of the program.
func runTest(t *testing.T, code string) ([]byte, []byte, error) {
	chk := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	gopath, err := ioutil.TempDir("", "profile-gopath")
	chk(err)
	defer os.RemoveAll(gopath)

	srcdir := filepath.Join(gopath, "src")
	err = os.Mkdir(srcdir, 0755)
	chk(err)
	src := filepath.Join(srcdir, "main.go")
	err = ioutil.WriteFile(src, []byte(code), 0644)
	chk(err)

	cmd := exec.Command("go", "run", src)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}
