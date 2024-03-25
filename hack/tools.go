// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

// +build tools

package tools

import (
	"k8s.io/code-generator"

	// Needed for protoc
	"github.com/gogo/protobuf/proto"
	"k8s.io/code-generator/cmd/go-to-protobuf/protoc-gen-gogo"
	"github.com/gogo/protobuf/protoc-gen-gogo"
	"github.com/gogo/protobuf/protoc-gen-gofast"
	"golang.org/x/tools/cmd/goimports"

	// Needed for k8s protobuf gen
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	"k8s.io/apimachinery/pkg/apis/testapigroup/v1"
)
