// Copyright 2022 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package diff

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

var (
	stdin io.ReadWriter = os.Stdin
)

// Returns diff of two io.Reader using cmd diff tool
func Diff(current, newer io.Reader) ([]byte, error) {
	f1, err := writeTempFile(current)
	if err != nil {
		return nil, err
	}
	defer os.Remove(f1)

	f2, err := writeTempFile(newer)
	if err != nil {
		return nil, err
	}
	defer os.Remove(f2)

	data, err := exec.Command("diff", "-U0", "--label=current", f1, "--label=newer", f2).CombinedOutput()
	if len(data) > 0 {
		// diff exits with a non-zero status when the files don't match.
		// Ignore that failure as long as we get output.
		err = nil
	}

	return data, err
}

func ReplaceWithSudo(originalFile string, newerContent io.Reader) error {
	f1, err := writeTempFile(newerContent)
	if err != nil {
		return err
	}
	defer os.Remove(f1)

	output1, err1 := exec.Command("cp", f1, originalFile).CombinedOutput()
	if err1 != nil {
		// try with sudo
		cmd := exec.Command("sudo", "cp", f1, originalFile)
		localStderr := &bytes.Buffer{}
		cmd.Stderr = localStderr
		cmd.Stdin = stdin // handles password input
		err2 := cmd.Run()
		if err2 != nil {
			e, err3 := io.ReadAll(localStderr)
			return fmt.Errorf(
				`Could not "cp" the current file, even with sudo: %s (%s) | %s (%s%s)`,
				err1, string(output1), err2, strings.TrimSpace(string(e)), errOrEmpty(err3),
			)
		}
	}

	return nil
}

func writeTempFile(data io.Reader) (string, error) {
	file, err := ioutil.TempFile("", "")
	if err != nil {
		return "", err
	}
	bytes, err := io.ReadAll(data)
	if err != nil {
		return "", err
	}
	_, err = file.Write(bytes)
	if err1 := file.Close(); err == nil {
		err = err1
	}
	if err != nil {
		os.Remove(file.Name())
		return "", err
	}
	return file.Name(), nil
}

func errOrEmpty(e error) string {
	if e == nil {
		return ""
	}
	return e.Error()
}
