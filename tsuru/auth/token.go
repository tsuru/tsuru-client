package auth

import (
	"errors"
	"sync"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/fs"
)

func writeToken(token string) error {
	tokenPaths := []string{
		cmd.JoinWithUserDir(".tsuru", "token"),
	}
	targetLabel, err := cmd.GetTargetLabel()
	if err == nil {
		err := filesystem().MkdirAll(cmd.JoinWithUserDir(".tsuru", "token.d"), 0700)
		if err != nil {
			return err
		}
		tokenPaths = append(tokenPaths, cmd.JoinWithUserDir(".tsuru", "token.d", targetLabel))
	}
	for _, tokenPath := range tokenPaths {
		file, err := filesystem().Create(tokenPath)
		if err != nil {
			return err
		}
		defer file.Close()
		n, err := file.WriteString(token)
		if err != nil {
			return err
		}
		if n != len(token) {
			return errors.New("failed to write token file.")
		}
	}
	return nil
}

var (
	fsystem   fs.Fs
	fsystemMu sync.Mutex
)

func filesystem() fs.Fs {
	fsystemMu.Lock()
	defer fsystemMu.Unlock()
	if fsystem == nil {
		fsystem = fs.OsFs{}
	}
	return fsystem
}
