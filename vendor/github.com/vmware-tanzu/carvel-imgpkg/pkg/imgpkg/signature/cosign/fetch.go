// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

// Using this code as is from: https://github.com/sigstore/cosign/blob/06657c5c2de996809216c2e4f17b70ae13d042da/pkg/cosign/fetch.go

//
// Copyright 2021 The Sigstore Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cosign

import (
	"strings"

	v1 "github.com/google/go-containerregistry/pkg/v1"
)

func Munge(desc v1.Descriptor) string {
	return signatureImageTagForDigest(desc.Digest.String())
}

func signatureImageTagForDigest(digest string) string {
	// sha256:... -> sha256-...
	return strings.ReplaceAll(digest, ":", "-") + ".sig"
}
