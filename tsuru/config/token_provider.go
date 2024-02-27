// Copyright 2024 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

type TokenProvider interface {
	// Token returns the content of header authorization, useful to propagate token to third parties such as plugins and websocket
	Token() (string, error)
}

var DefaultTokenProvider TokenProvider = TokenProviderV1()

type tokenProviderV1 struct{}

func (t *tokenProviderV1) Token() (string, error) {
	return ReadTokenV1()
}

func TokenProviderV1() TokenProvider {
	return &tokenProviderV1{}
}
