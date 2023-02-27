// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/google/go-containerregistry/pkg/authn"
	regname "github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
)

// NewMultiRoundTripperStorage Creates a struct that holds RoundTripper
func NewMultiRoundTripperStorage(baseRoundTripper http.RoundTripper) *MultiRoundTripperStorage {
	return &MultiRoundTripperStorage{
		baseRoundTripper: baseRoundTripper,
		readWriteAccess:  &sync.Mutex{},
		transports:       map[string]map[string]map[string]http.RoundTripper{},
	}
}

// NewSingleTripperStorage Creates a struct that holds RoundTripper
func NewSingleTripperStorage(baseRoundTripper http.RoundTripper) *SingleTripperStorage {
	return &SingleTripperStorage{
		baseRoundTripper: baseRoundTripper,
		readWriteAccess:  &sync.Mutex{},
	}
}

// NewNoopRoundTripperStorage Creates a struct that does not save any RoundTripper
func NewNoopRoundTripperStorage() *NoopRoundTripperStorage {
	return &NoopRoundTripperStorage{}
}

// MultiRoundTripperStorage Maintains a storage of all the available RoundTripper for different registries and repositories
type MultiRoundTripperStorage struct {
	baseRoundTripper http.RoundTripper
	transports       map[string]map[string]map[string]http.RoundTripper
	readWriteAccess  *sync.Mutex
}

// BaseRoundTripper retrieves the base RoundTripper used by the store
func (r MultiRoundTripperStorage) BaseRoundTripper() http.RoundTripper {
	return r.baseRoundTripper
}

// RoundTripper Retrieve the RoundTripper to be used for a particular registry and repository or nil if it cannot be found
func (r *MultiRoundTripperStorage) RoundTripper(repo regname.Repository, scope string) http.RoundTripper {
	r.readWriteAccess.Lock()
	defer r.readWriteAccess.Unlock()

	s := strings.Split(scope, ":")
	if len(s) != 3 {
		panic(fmt.Sprintf("Internal inconsistency: expected scope '%s' to have 3 fields", scope))
	}
	// Maybe we should check to make sure only 1 repository is present in the scopes
	method := s[2]

	if _, ok := r.transports[repo.RegistryStr()]; !ok {
		return nil
	}

	if _, ok := r.transports[repo.RegistryStr()][repo.RepositoryStr()]; !ok {
		return nil
	}

	if _, ok := r.transports[repo.RegistryStr()][repo.RepositoryStr()][method]; !ok {
		if method == transport.PullScope {
			if _, ok := r.transports[repo.RegistryStr()][repo.RepositoryStr()][transport.PushScope]; ok {
				return r.transports[repo.RegistryStr()][repo.RepositoryStr()][transport.PushScope]
			}
		}
		return nil
	}

	return r.transports[repo.RegistryStr()][repo.RepositoryStr()][method]
}

// CreateRoundTripper Creates a new RoundTripper
// scope field has the following format "repository:/org/suborg/repo_name:pull,push"
//
//	for more information check https://github.com/distribution/distribution/blob/263da70ea6a4e96f61f7a6770273ec6baac38941/docs/spec/auth/token.md#requesting-a-token
func (r *MultiRoundTripperStorage) CreateRoundTripper(reg regname.Registry, auth authn.Authenticator, scope string) (http.RoundTripper, error) {
	r.readWriteAccess.Lock()
	defer r.readWriteAccess.Unlock()

	rt, err := transport.NewWithContext(context.Background(), reg, auth, r.baseRoundTripper, []string{scope})
	if err != nil {
		return nil, fmt.Errorf("Unable to create round tripper: %s", err)
	}

	if _, ok := r.transports[reg.RegistryStr()]; !ok {
		r.transports[reg.RegistryStr()] = map[string]map[string]http.RoundTripper{}
	}
	s := strings.Split(scope, ":")
	if len(s) != 3 {
		panic(fmt.Sprintf("Internal inconsistency: expected scope '%s' to have 3 fields", scope))
	}
	// Maybe we should check to make sure only 1 repository is present in the scopes
	repository := s[1]
	method := s[2]

	if _, ok := r.transports[reg.RegistryStr()][repository]; !ok {
		r.transports[reg.RegistryStr()][repository] = map[string]http.RoundTripper{}
	}

	r.transports[reg.RegistryStr()][repository][method] = rt

	return rt, nil
}

// SingleTripperStorage Maintains a storage of all the available RoundTripper for different registries and repositories
type SingleTripperStorage struct {
	baseRoundTripper http.RoundTripper
	transport        http.RoundTripper
	readWriteAccess  *sync.Mutex
}

// RoundTripper Retrieve the RoundTripper to be used for a particular registry and repository or nil if it cannot be found
func (r *SingleTripperStorage) RoundTripper(_ regname.Repository, _ string) http.RoundTripper {
	r.readWriteAccess.Lock()
	defer r.readWriteAccess.Unlock()

	return r.transport
}

// BaseRoundTripper retrieves the base RoundTripper used by the store
func (r SingleTripperStorage) BaseRoundTripper() http.RoundTripper {
	return r.baseRoundTripper
}

// CreateRoundTripper Creates a new RoundTripper
// scope field has the following format "repository:/org/suborg/repo_name:pull,push"
//
//	for more information check https://github.com/distribution/distribution/blob/263da70ea6a4e96f61f7a6770273ec6baac38941/docs/spec/auth/token.md#requesting-a-token
func (r *SingleTripperStorage) CreateRoundTripper(reg regname.Registry, auth authn.Authenticator, scope string) (http.RoundTripper, error) {
	r.readWriteAccess.Lock()
	defer r.readWriteAccess.Unlock()

	rt, err := transport.NewWithContext(context.Background(), reg, auth, r.baseRoundTripper, []string{scope})
	if err != nil {
		return nil, fmt.Errorf("Unable to create round tripper: %s", err)
	}

	r.transport = rt

	return rt, nil
}

// NoopRoundTripperStorage does not store any http.RoundTripper
type NoopRoundTripperStorage struct{}

// RoundTripper returns nil to all invocations
func (n NoopRoundTripperStorage) RoundTripper(regname.Repository, string) http.RoundTripper {
	return nil
}

// CreateRoundTripper does nothing
func (n NoopRoundTripperStorage) CreateRoundTripper(reg regname.Registry, auth authn.Authenticator, scope string) (http.RoundTripper, error) {
	return nil, nil
}

// BaseRoundTripper returns nil to all invocations
func (n NoopRoundTripperStorage) BaseRoundTripper() http.RoundTripper {
	return nil
}
