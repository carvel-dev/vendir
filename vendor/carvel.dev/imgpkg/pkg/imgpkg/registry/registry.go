// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"sync"
	"time"

	"carvel.dev/imgpkg/pkg/imgpkg/internal/util"
	"carvel.dev/imgpkg/pkg/imgpkg/registry/auth"
	regauthn "github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/logs"
	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	regremote "github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
)

type Opts struct {
	CACertPaths []string
	VerifyCerts bool
	Insecure    bool

	IncludeNonDistributableLayers bool

	Username string
	Password string
	Token    string
	Anon     bool

	EnableIaasAuthProviders bool

	ResponseHeaderTimeout time.Duration
	RetryCount            int

	EnvironFunc     func() []string
	ActiveKeychains []auth.IAASKeychain

	SessionID string
}

// DeepCopy the options to a new struct
func (o Opts) DeepCopy() Opts {
	result := Opts{
		VerifyCerts:                   o.VerifyCerts,
		Insecure:                      o.Insecure,
		IncludeNonDistributableLayers: o.IncludeNonDistributableLayers,
		Username:                      o.Username,
		Password:                      o.Password,
		Token:                         o.Token,
		Anon:                          o.Anon,
		EnableIaasAuthProviders:       o.EnableIaasAuthProviders,
		ResponseHeaderTimeout:         o.ResponseHeaderTimeout,
		RetryCount:                    o.RetryCount,
		EnvironFunc:                   o.EnvironFunc,
	}
	for _, path := range o.CACertPaths {
		result.CACertPaths = append(result.CACertPaths, path)
	}
	for _, keychain := range o.ActiveKeychains {
		result.ActiveKeychains = append(result.ActiveKeychains, keychain)
	}
	return result
}

// Registry Interface to access the registry
type Registry interface {
	Get(reference regname.Reference) (*regremote.Descriptor, error)
	Digest(reference regname.Reference) (regv1.Hash, error)
	Index(reference regname.Reference) (regv1.ImageIndex, error)
	Image(reference regname.Reference) (regv1.Image, error)
	FirstImageExists(digests []string) (string, error)

	MultiWrite(imageOrIndexesToUpload map[regname.Reference]regremote.Taggable, concurrency int, updatesCh chan regv1.Update) error
	WriteImage(regname.Reference, regv1.Image, chan regv1.Update) error
	WriteIndex(reference regname.Reference, index regv1.ImageIndex) error
	WriteTag(tag regname.Tag, taggable regremote.Taggable) error

	ListTags(repo regname.Repository) ([]string, error)

	CloneWithSingleAuth(imageRef regname.Tag) (Registry, error)
	CloneWithLogger(logger util.ProgressLogger) Registry
}

// ImagesReader Interface for Reading Images
//
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . ImagesReader
type ImagesReader interface {
	Get(regname.Reference) (*regremote.Descriptor, error)
	Digest(regname.Reference) (regv1.Hash, error)
	Index(regname.Reference) (regv1.ImageIndex, error)
	Image(regname.Reference) (regv1.Image, error)
	FirstImageExists(digests []string) (string, error)
}

// ImagesReaderWriter Interface for Reading and Writing Images
//
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . ImagesReaderWriter
type ImagesReaderWriter interface {
	ImagesReader
	MultiWrite(imageOrIndexesToUpload map[regname.Reference]regremote.Taggable, concurrency int, updatesCh chan regv1.Update) error
	WriteImage(regname.Reference, regv1.Image, chan regv1.Update) error
	WriteIndex(regname.Reference, regv1.ImageIndex) error
	WriteTag(regname.Tag, regremote.Taggable) error

	CloneWithSingleAuth(imageRef regname.Tag) (Registry, error)
	CloneWithLogger(logger util.ProgressLogger) Registry
}

var _ Registry = &SimpleRegistry{}

// RoundTripperStorage Storage of RoundTripper that will be used to talk to the registry
type RoundTripperStorage interface {
	RoundTripper(repo regname.Repository, scope string) http.RoundTripper
	CreateRoundTripper(reg regname.Registry, auth regauthn.Authenticator, scope string) (http.RoundTripper, error)
	BaseRoundTripper() http.RoundTripper
}

// SimpleRegistry Implements Registry interface
type SimpleRegistry struct {
	remoteOpts      []regremote.Option
	refOpts         []regname.Option
	keychain        regauthn.Keychain
	authn           map[string]regauthn.Authenticator
	roundTrippers   RoundTripperStorage
	transportAccess *sync.Mutex
}

// NewBasicRegistry does not provide any special behavior and all the options as passed as is to the underlying library
func NewBasicRegistry(regOpts ...regremote.Option) (*SimpleRegistry, error) {
	return &SimpleRegistry{
		remoteOpts:      regOpts,
		roundTrippers:   NewNoopRoundTripperStorage(),
		transportAccess: &sync.Mutex{},
	}, nil
}

// NewSimpleRegistry Builder for a Simple Registry
func NewSimpleRegistry(opts Opts) (*SimpleRegistry, error) {
	httpTran, err := newHTTPTransport(opts)
	if err != nil {
		return nil, fmt.Errorf("Creating registry HTTP transport: %s", err)
	}
	return NewSimpleRegistryWithTransport(opts, httpTran)
}

// NewSimpleRegistryWithTransport Creates a new Simple Registry using the provided transport
func NewSimpleRegistryWithTransport(opts Opts, rTripper http.RoundTripper) (*SimpleRegistry, error) {
	var refOpts []regname.Option
	if opts.Insecure {
		refOpts = append(refOpts, regname.Insecure)
	}

	keychain, err := Keychain(
		auth.KeychainOpts{
			Username:                opts.Username,
			Password:                opts.Password,
			Token:                   opts.Token,
			Anon:                    opts.Anon,
			EnableIaasAuthProviders: opts.EnableIaasAuthProviders,
			ActiveKeychains:         opts.ActiveKeychains,
		},
		opts.EnvironFunc,
	)
	if err != nil {
		return nil, fmt.Errorf("Creating registry keychain: %s", err)
	}

	var regRemoteOptions []regremote.Option
	if opts.IncludeNonDistributableLayers {
		regRemoteOptions = append(regRemoteOptions, regremote.WithNondistributable)
	}
	tries := opts.RetryCount
	if tries == 0 {
		tries = 1
	}

	retryBackoff := regremote.Backoff{
		Duration: 100 * time.Millisecond,
		Factor:   2,
		Jitter:   0,
		Steps:    tries,
		Cap:      1 * time.Second,
	}
	regRemoteOptions = append(regRemoteOptions, regremote.WithRetryBackoff(retryBackoff))

	baseRoundTripper := rTripper
	if logs.Enabled(logs.Debug) {
		baseRoundTripper = transport.NewLogger(rTripper)
	}

	sessionID := opts.SessionID
	if sessionID == "" {
		sessionID = fmt.Sprint(rand.Intn(9999999999))
	}
	baseRoundTripper = NewImgpkgRoundTripper(baseRoundTripper, sessionID)

	// Wrap the transport in something that can retry network flakes.
	baseRoundTripper = transport.NewRetry(baseRoundTripper, transport.WithRetryBackoff(retryBackoff))

	return &SimpleRegistry{
		remoteOpts:      regRemoteOptions,
		refOpts:         refOpts,
		keychain:        keychain,
		roundTrippers:   NewMultiRoundTripperStorage(baseRoundTripper),
		authn:           map[string]regauthn.Authenticator{},
		transportAccess: &sync.Mutex{},
	}, nil
}

// CloneWithSingleAuth produces a copy of this Registry whose keychain has exactly one auth — the one that can be used
// to access imageRef. If no keychain is explicitly configured on this Registry, the copy is a BasicRegistry.
// A Registry need to be provided as the first parameter or the function will panic
func (r SimpleRegistry) CloneWithSingleAuth(imageRef regname.Tag) (Registry, error) {
	if r.keychain == nil { // If no keychain is present it assumes NewBasicRegistry was used to create the Registry. So we short circuit this execution
		return NewBasicRegistry(r.remoteOpts...)
	}

	imgAuth, err := r.keychain.Resolve(imageRef)
	if err != nil {
		return nil, err
	}

	keychain := auth.NewSingleAuthKeychain(imgAuth)
	rt := r.roundTrippers.RoundTripper(imageRef.Repository, imageRef.Scope(transport.PullScope))
	if rt == nil {
		rt = r.roundTrippers.BaseRoundTripper()
	}

	var singleRt RoundTripperStorage = NewNoopRoundTripperStorage()
	if rt != nil {
		singleRt = NewSingleTripperStorage(rt)
	}

	return &SimpleRegistry{
		remoteOpts:      r.remoteOpts,
		refOpts:         r.refOpts,
		keychain:        keychain,
		roundTrippers:   singleRt,
		authn:           map[string]regauthn.Authenticator{},
		transportAccess: &sync.Mutex{},
	}, nil
}

// CloneWithLogger Clones the provided registry updating the progress logger to NoTTYLogger
// that does not display the progress bar
func (r SimpleRegistry) CloneWithLogger(_ util.ProgressLogger) Registry {
	return &SimpleRegistry{
		remoteOpts:      r.remoteOpts,
		refOpts:         r.refOpts,
		keychain:        r.keychain,
		roundTrippers:   r.roundTrippers,
		authn:           map[string]regauthn.Authenticator{},
		transportAccess: &sync.Mutex{},
	}
}

// readOpts Returns the readOpts + the keychain
func (r *SimpleRegistry) readOpts(ref regname.Reference) ([]regremote.Option, error) {
	rt, authn, err := r.transport(ref, ref.Scope(transport.PullScope))
	if err != nil {
		return nil, err
	}
	return append([]regremote.Option{regremote.WithAuth(authn), regremote.WithTransport(rt)}, r.remoteOpts...), nil
}

// writeOpts Returns the writeOpts + the keychain
func (r *SimpleRegistry) writeOpts(ref regname.Reference) ([]regremote.Option, error) {
	rt, auth, err := r.transport(ref, ref.Scope(transport.PushScope))
	if err != nil {
		return nil, err
	}

	return append([]regremote.Option{regremote.WithAuth(auth), regremote.WithTransport(rt)}, r.remoteOpts...), nil
}

// transport Retrieve the RoundTripper that can be used to access the repository
func (r *SimpleRegistry) transport(ref regname.Reference, scope string) (http.RoundTripper, regauthn.Authenticator, error) {
	registry := ref.Context()
	registryKey := registry.Name()
	// The idea is that we can only retrieve 1 RoundTripper at a time to ensure that we do not create
	// the same RoundTripper multiple times
	r.transportAccess.Lock()
	defer r.transportAccess.Unlock()
	if r.authn == nil {
		// This shouldn't happen, but let's defend in depth
		r.authn = map[string]regauthn.Authenticator{}
	}

	rt := r.roundTrippers.RoundTripper(registry, scope)
	if rt == nil {
		if r.keychain == nil {
			return nil, nil, nil
		}

		resolvedAuth, err := r.keychain.Resolve(registry)
		if err != nil {
			return nil, nil, fmt.Errorf("Unable retrieve credentials for registry: %s", err)
		}
		r.authn[registryKey] = resolvedAuth
		rt, err = r.roundTrippers.CreateRoundTripper(registry.Registry, resolvedAuth, scope)
		if err != nil {
			return nil, nil, fmt.Errorf("Error while preparing a transport to talk with the registry: %s", err)
		}
	}

	auth, ok := r.authn[registryKey]
	if !ok {
		regauth, err := r.keychain.Resolve(registry)
		if err != nil {
			return nil, nil, fmt.Errorf("Unable to get authenticator for registry %s: %s", ref.Context().Name(), err)
		}
		r.authn[registryKey] = regauth
		auth = regauth
	}

	return rt, auth, nil
}

// Get Retrieve Image descriptor for an Image reference
func (r *SimpleRegistry) Get(ref regname.Reference) (*regremote.Descriptor, error) {
	if err := r.validateRef(ref); err != nil {
		return nil, err
	}
	overriddenRef, err := regname.ParseReference(ref.String(), r.refOpts...)
	if err != nil {
		return nil, err
	}
	opts, err := r.readOpts(overriddenRef)
	if err != nil {
		return nil, err
	}
	return regremote.Get(overriddenRef, opts...)
}

// Digest Retrieve the Digest for an Image reference
func (r *SimpleRegistry) Digest(ref regname.Reference) (regv1.Hash, error) {
	if err := r.validateRef(ref); err != nil {
		return regv1.Hash{}, err
	}
	overriddenRef, err := regname.ParseReference(ref.String(), r.refOpts...)
	if err != nil {
		return regv1.Hash{}, err
	}

	opts, err := r.readOpts(overriddenRef)
	if err != nil {
		return regv1.Hash{}, err
	}
	desc, err := regremote.Head(overriddenRef, opts...)
	if err != nil {
		getDesc, err := regremote.Get(overriddenRef, opts...)
		if err != nil {
			return regv1.Hash{}, err
		}
		return getDesc.Digest, nil
	}

	return desc.Digest, nil
}

// Image Retrieve the regv1.Image struct for an Image reference
func (r *SimpleRegistry) Image(ref regname.Reference) (regv1.Image, error) {
	if err := r.validateRef(ref); err != nil {
		return nil, err
	}
	overriddenRef, err := regname.ParseReference(ref.String(), r.refOpts...)
	if err != nil {
		return nil, err
	}

	opts, err := r.readOpts(overriddenRef)
	if err != nil {
		return nil, err
	}
	return regremote.Image(overriddenRef, opts...)
}

// MultiWrite Upload multiple Images in Parallel to the Registry
func (r *SimpleRegistry) MultiWrite(imageOrIndexesToUpload map[regname.Reference]regremote.Taggable, concurrency int, updatesCh chan regv1.Update) error {
	overriddenImageOrIndexesToUploadRef := map[regname.Reference]regremote.Taggable{}

	var singleRef regname.Reference
	for ref, taggable := range imageOrIndexesToUpload {
		if err := r.validateRef(ref); err != nil {
			return err
		}
		overriddenRef, err := regname.ParseReference(ref.String(), r.refOpts...)
		if err != nil {
			return err
		}
		singleRef = overriddenRef

		overriddenImageOrIndexesToUploadRef[overriddenRef] = taggable
	}

	opts, err := r.writeOpts(singleRef)
	if err != nil {
		return err
	}
	rOpts := append(append([]regremote.Option{}, opts...), regremote.WithJobs(concurrency))
	if updatesCh != nil {
		rOpts = append(rOpts, regremote.WithProgress(updatesCh))
	}
	return regremote.MultiWrite(overriddenImageOrIndexesToUploadRef, rOpts...)
}

// WriteImage Upload Image to registry
func (r *SimpleRegistry) WriteImage(ref regname.Reference, img regv1.Image, updatesCh chan regv1.Update) error {
	if err := r.validateRef(ref); err != nil {
		return err
	}
	overriddenRef, err := regname.ParseReference(ref.String(), r.refOpts...)
	if err != nil {
		return err
	}

	opts, err := r.writeOpts(overriddenRef)
	if err != nil {
		return err
	}
	if updatesCh != nil {
		opts = append(opts, regremote.WithProgress(updatesCh))
	}
	err = regremote.Write(overriddenRef, img, opts...)
	if err != nil {
		return fmt.Errorf("Writing image: %s", err)
	}

	return nil
}

// Index Retrieve regv1.ImageIndex struct for an Index reference
func (r *SimpleRegistry) Index(ref regname.Reference) (regv1.ImageIndex, error) {
	if err := r.validateRef(ref); err != nil {
		return nil, err
	}
	overriddenRef, err := regname.ParseReference(ref.String(), r.refOpts...)
	if err != nil {
		return nil, err
	}
	opts, err := r.readOpts(overriddenRef)
	if err != nil {
		return nil, err
	}
	return regremote.Index(overriddenRef, opts...)
}

// WriteIndex Uploads the Index manifest to the registry
func (r *SimpleRegistry) WriteIndex(ref regname.Reference, idx regv1.ImageIndex) error {
	if err := r.validateRef(ref); err != nil {
		return err
	}
	overriddenRef, err := regname.ParseReference(ref.String(), r.refOpts...)
	if err != nil {
		return err
	}

	opts, err := r.writeOpts(overriddenRef)
	if err != nil {
		return err
	}

	err = regremote.WriteIndex(overriddenRef, idx, opts...)
	if err != nil {
		return fmt.Errorf("Writing image index: %s", err)
	}

	return nil
}

// WriteTag Tag the referenced Image
func (r *SimpleRegistry) WriteTag(ref regname.Tag, taggagle regremote.Taggable) error {
	if err := r.validateRef(ref); err != nil {
		return err
	}
	overriddenRef, err := regname.NewTag(ref.String(), r.refOpts...)
	if err != nil {
		return err
	}

	opts, err := r.writeOpts(overriddenRef)
	if err != nil {
		return err
	}

	err = regremote.Tag(overriddenRef, taggagle, opts...)
	if err != nil {
		return fmt.Errorf("Tagging image: %s", err)
	}

	return nil
}

// ListTags Retrieve all tags associated with a Repository
func (r *SimpleRegistry) ListTags(repo regname.Repository) ([]string, error) {
	overriddenRepo, err := regname.NewRepository(repo.Name(), r.refOpts...)
	if err != nil {
		return nil, err
	}
	repoRef, err := regname.ParseReference(overriddenRepo.String(), r.refOpts...)
	if err != nil {
		return nil, err
	}
	opts, err := r.readOpts(repoRef)
	if err != nil {
		return nil, err
	}

	return regremote.List(overriddenRepo, opts...)
}

// FirstImageExists Returns the first of the provided Image Digests that exists in the Registry
func (r *SimpleRegistry) FirstImageExists(digests []string) (string, error) {
	var err error
	for _, img := range digests {
		ref, parseErr := regname.NewDigest(img)
		if parseErr != nil {
			return "", parseErr
		}
		_, err = r.Digest(ref)
		if err == nil {
			return img, nil
		}
	}
	return "", fmt.Errorf("Checking image existence: %s", err)
}

func newHTTPTransport(opts Opts) (*http.Transport, error) {
	var pool *x509.CertPool

	var err error
	pool, err = x509.SystemCertPool()
	if err != nil {
		return nil, err
	}

	if len(opts.CACertPaths) > 0 {
		for _, path := range opts.CACertPaths {
			if certs, err := os.ReadFile(path); err != nil {
				return nil, fmt.Errorf("Reading CA certificates from '%s': %s", path, err)
			} else if ok := pool.AppendCertsFromPEM(certs); !ok {
				return nil, fmt.Errorf("Adding CA certificates from '%s': failed", path)
			}
		}
	}

	clonedDefaultTransport := http.DefaultTransport.(*http.Transport).Clone()
	clonedDefaultTransport.ForceAttemptHTTP2 = false
	clonedDefaultTransport.ResponseHeaderTimeout = opts.ResponseHeaderTimeout
	clonedDefaultTransport.TLSClientConfig = &tls.Config{
		RootCAs:            pool,
		InsecureSkipVerify: opts.VerifyCerts == false,
	}

	return clonedDefaultTransport, nil
}

var protocolMatcher = regexp.MustCompile(`\Ahttps?://`)

func (SimpleRegistry) validateRef(ref regname.Reference) error {
	if match := protocolMatcher.FindString(ref.String()); len(match) > 0 {
		return fmt.Errorf("Reference '%s' should not include %s protocol prefix", ref, match)
	}
	return nil
}
