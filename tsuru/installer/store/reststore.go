// Copyright 2016 tsuru-client authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package store

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/docker/machine/libmachine/host"
)

type Doer interface {
	Do(request *http.Request) (*http.Response, error)
}

type RestStore struct {
	URL  string
	doer Doer
}

func NewRestStore(url string, doer Doer) *RestStore {
	if doer == nil {
		doer = &http.Client{}
	}
	return &RestStore{
		URL:  url,
		doer: doer,
	}
}

func (s *RestStore) Exists(name string) (bool, error) {
	req, err := http.NewRequest("HEAD", s.URL+name, nil)
	if err != nil {
		return false, err
	}
	resp, err := s.doer.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	return false, nil
}

func (s *RestStore) List() ([]string, error) {
	req, err := http.NewRequest("GET", s.URL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.doer.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var hosts []*host.Host
	err = json.Unmarshal(result, &hosts)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, h := range hosts {
		names = append(names, h.Name)
	}
	return names, nil
}

func (s *RestStore) Load(name string) (*host.Host, error) {
	req, err := http.NewRequest("GET", s.URL+name, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.doer.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var host *host.Host
	err = json.Unmarshal(result, &host)
	if err != nil {
		return nil, err
	}
	return host, nil
}

func (s *RestStore) Remove(name string) error {
	req, err := http.NewRequest("DELETE", s.URL+name, nil)
	if err != nil {
		return err
	}
	_, err = s.doer.Do(req)
	return err
}

func (s *RestStore) Save(host *host.Host) error {
	b, err := json.Marshal(host)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", s.URL+host.Name, strings.NewReader(string(b)))
	if err != nil {
		return err
	}
	_, err = s.doer.Do(req)
	return err
}
