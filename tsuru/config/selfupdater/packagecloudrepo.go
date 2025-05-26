// Copyright 2022 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package selfupdater

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/tsuru/tsuru-client/tsuru/config/diff"
	"github.com/tsuru/tsuru/fs"
)

const (
	debRE string = `(?P<pre>^deb(-src)?.* )(?P<url>https://packagecloud\.io/tsuru/\w+/)(?P<os>\w+)(?P<sep>/? )(?P<dist>[0-9A-Za-z.]+)(?P<end> main.*$)`
	rpmRE string = `(?P<pre>^baseurl=)(?P<url>https://packagecloud\.io/tsuru/\w+/)(?P<os>\w+)(?P<sep>/)(?P<dist>[0-9A-Za-z.]+)(?P<end>/.*$)`
)

var (
	stdin   io.ReadWriter = os.Stdin
	fsystem fs.Fs         = fs.OsFs{}
)

func reFindSubmatchMap(r *regexp.Regexp, data string) map[string]string {
	match := r.FindStringSubmatch(data)
	matchMap := make(map[string]string)
	for i, name := range r.SubexpNames() {
		if i != 0 && i < len(match) {
			matchMap[name] = match[i]
		}
	}
	return matchMap
}

func findConfRepoPath() (repoType string, filepath string) {
	if _, err := fsystem.Stat("/etc/apt/sources.list.d/tsuru_stable.list"); err == nil {
		return "deb", "/etc/apt/sources.list.d/tsuru_stable.list"
	}
	if _, err := fsystem.Stat("/etc/zypp/repos.d/tsuru_stable.repo"); err == nil {
		return "rpm", "/etc/zypp/repos.d/tsuru_stable.repo"
	}
	if _, err := fsystem.Stat("/etc/yum.repos.d/tsuru_stable.repo"); err == nil {
		return "rpm", "/etc/yum.repos.d/tsuru_stable.repo"
	}
	return "", ""
}

// replaceConfLine checks line with regex r.
func replaceConfLine(r *regexp.Regexp, line string) (wasReplaced bool, replacedLine string) {
	m := reFindSubmatchMap(r, line)
	if len(m) > 0 {
		for _, k := range []string{"pre", "url", "os", "sep", "dist", "end"} {
			if v := m[k]; v == "" {
				return false, line
			}
		}
		if m["os"] != "any" || m["dist"] != "any" {
			// was:         pre   +    url   +  os   +    sep   + dist  +    end
			return true, m["pre"] + m["url"] + "any" + m["sep"] + "any" + m["end"]
		}
	}
	return false, line
}

func replaceConf(r *regexp.Regexp, reader io.Reader) (hasDiff bool, replacedContent []byte, err error) {
	scanner := bufio.NewScanner(reader)
	writer := &bytes.Buffer{}
	for scanner.Scan() {
		wasReplaced, line := replaceConfLine(r, scanner.Text())
		if wasReplaced {
			hasDiff = true
		}
		writer.WriteString(line + "\n")
	}
	if err = scanner.Err(); err != nil {
		return hasDiff, writer.Bytes(), fmt.Errorf("got error on scanning repoConfPath lines: %w", err)
	}
	return hasDiff, writer.Bytes(), err
}

func checkUpToDateConfRepo(repoType, repoConfPath string) error {
	var r *regexp.Regexp

	if ignoreOutdatedRepoStr := os.Getenv("TSURU_CLIENT_IGNORE_OUTDATED_PCLOUD_REPO"); ignoreOutdatedRepoStr != "" {
		if ignoreOutdatedRepo, err := strconv.ParseBool(ignoreOutdatedRepoStr); err != nil {
			fmt.Fprintln(stderr, "WARN: when setting TSURU_CLIENT_IGNORE_OUTDATED_PCLOUD_REPO, it must be either true or false")
		} else if ignoreOutdatedRepo {
			return nil
		}
	}

	switch repoType {
	case "deb":
		r = regexp.MustCompile(debRE)
	case "rpm":
		r = regexp.MustCompile(rpmRE)
	default:
		return nil
	}

	// Getting original content
	originalF, err := fsystem.Open(repoConfPath)
	if err != nil {
		return fmt.Errorf("could not open repoConfPath: %w", err)
	}
	originalData, err := io.ReadAll(originalF)
	if err != nil {
		return fmt.Errorf("could not read repoConfPath: %w", err)
	}

	// Detecting diff
	hasDiff, newContent, err := replaceConf(r, bytes.NewReader(originalData))
	if err != nil {
		return fmt.Errorf("could not replaceConf: %w", err)
	}
	if !hasDiff {
		return nil
	}

	if _, ok := os.LookupEnv("TSURU_TOKEN"); ok {
		// Using TSURU_TOKEN probably means running tsuru-client from a script. Print a warning and exit with no error
		fmt.Fprintf(stderr, "Warn: %q is using an outdated repository. Use any/any instead of os/dist\n", repoConfPath)
		return nil
	}

	// Printing info about what is going on
	fmt.Fprintf(stderr, "\n")
	fmt.Fprintf(stderr, "\n")
	fmt.Fprintf(stderr, "############## Breaking change ##############\n")
	fmt.Fprintf(stderr, "  Tsuru-client appears to have been installed with a package manager (.%s)\n", repoType)
	fmt.Fprintf(stderr, "  The packagecloud repository in-use suffered a breaking change (applied with tsuru-client:1.13.0)\n")
	fmt.Fprintf(stderr, "  In order to receive future updates, the repo must be updated to use the any/any format (instead of os/distro)\n")
	fmt.Fprintf(stderr, "  You may update it later and ignore this warning by setting the env TSURU_CLIENT_IGNORE_OUTDATED_PCLOUD_REPO=t\n\n")

	// Asking for auto changes
	answ := ""
	scanner := bufio.NewScanner(stdin)

	for try := 0; answ != "yes" && answ != "no"; try++ {
		if try >= 10 {
			return fmt.Errorf("asked too many questions. Modify %q manually", repoConfPath)
		}
		fmt.Fprintf(stderr, "  Do you want to override the content now (or show the diff)? (will ask sudo password) [yes/no/diff] ")
		scanner.Scan()
		err1 := scanner.Err()
		if err1 != nil {
			return err1
		}
		answ = strings.TrimSpace(scanner.Text())

		if answ == "diff" {
			fmt.Fprintf(stderr, "  Check the diff for %q:\n", repoConfPath)
			fmt.Fprintf(stderr, "#-----------------------------------------------------------------------#\n")
			dif, err2 := diff.Diff(bytes.NewReader(originalData), bytes.NewReader(newContent))
			if err2 != nil {
				fmt.Fprintf(stderr, "    >> Got error executing diff: %v \n", err)
			} else {
				fmt.Fprintf(stderr, "%v", string(dif))
			}
			fmt.Fprintf(stderr, "#-----------------------------------------------------------------------#\n")
		}
	}
	if answ != "yes" {
		return nil
	}

	// Replacing the files
	fmt.Fprintf(stderr, "  Writing %q (sudo password may be required) ...\n", repoConfPath)
	if err1 := diff.ReplaceWithSudo(repoConfPath, bytes.NewReader(newContent)); err1 != nil {
		return err1
	}

	// Verifying the replacement
	replacedF, err := fsystem.Open(repoConfPath)
	if err != nil {
		return fmt.Errorf("replacement could not be confirmed: %w", err)
	}
	replacedData, err := io.ReadAll(replacedF)
	if err != nil {
		return fmt.Errorf("replacement could not be confirmed: %w", err)
	}
	hasDiff, _, err = replaceConf(r, bytes.NewReader(replacedData))
	if err != nil {
		return fmt.Errorf("replacement could not be confirmed: %w", err)
	}
	if hasDiff {
		fmt.Fprintln(stderr, "conf replacement was not succeeded :(")
	} else {
		fmt.Fprintln(stderr, "Succeed!")
	}

	return nil
}

func CheckPackageCloudRepo() error {
	repoType, repoConfPath := findConfRepoPath()
	return checkUpToDateConfRepo(repoType, repoConfPath)
}
