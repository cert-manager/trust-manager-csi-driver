/*
Copyright 2024 The cert-manager Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package state

import (
	"bytes"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// ObjectEncoder is used to encode an object for storage
type ObjectEncoder[T any] interface {
	// Encode will encode the object for storage, returning a byte slice that
	// can be understood by the decode method.
	Encode(T) ([]byte, error)
	// Decode will decode the object, returning the object.
	Decode([]byte) (T, error)
}

// NewVersionedObjectEncoder implements an ObjectLoader that accepts an internal
// object type, but converts it to a versioned object before encoding.
//
// This allows the schema to evolve and change while still being able to load
// older files.
func NewVersionedObjectEncoder[
	InternalVersion any,
	StorageVersion any,
	IP ObjectPtr[InternalVersion],
	SP ObjectPtr[StorageVersion],
](scheme *runtime.Scheme) (ObjectEncoder[InternalVersion], error) {
	return &versionedObjectEncoder[InternalVersion, StorageVersion, IP, SP]{
		scheme:     scheme,
		serializer: json.NewSerializerWithOptions(json.DefaultMetaFactory, scheme, scheme, json.SerializerOptions{Yaml: true, Pretty: true}),
	}, nil
}

type versionedObjectEncoder[InternalVersion, StorageVersion any, IP ObjectPtr[InternalVersion], SP ObjectPtr[StorageVersion]] struct {
	scheme     *runtime.Scheme
	serializer *json.Serializer
}

func (e versionedObjectEncoder[I, S, IP, SP]) Encode(obj I) ([]byte, error) {
	// Convert from the internal object type to the versioned object that we
	// want to save
	var versioned S
	if err := e.scheme.Convert(&obj, SP(&versioned), nil); err != nil {
		return nil, fmt.Errorf("could not convert object to storage version: %w", err)
	}

	// Get Group/Version/Kind of the stored object, we want to ensure this is
	// set on the object
	gvk, err := apiutil.GVKForObject(SP(&versioned), e.scheme)
	if err != nil {
		return nil, fmt.Errorf("could not get group/version/kind of object: %w", err)
	}

	// Set the discovered Group/Version/Kind of the stored object.
	SP(&versioned).GetObjectKind().SetGroupVersionKind(gvk)

	var buffer bytes.Buffer
	if err := e.serializer.Encode(SP(&versioned), &buffer); err != nil {
		return nil, fmt.Errorf("could encode object as JSON: %w", err)
	}

	return buffer.Bytes(), nil
}

func (e versionedObjectEncoder[I, S, IP, SP]) Decode(data []byte) (I, error) {
	var internal I

	// Use the serializer to decode the []byte into an object, the serializer
	// uses the *runtime.Scheme to determine the object type to decode into.
	//
	// Due to this we have no guarantees of what type the resulting object
	// contains. This does not matter though as we are going to attempt to
	// convert it to the internal type.
	versioned, _, err := e.serializer.Decode(data, nil, nil)
	if err != nil {
		return internal, fmt.Errorf("could not decode object: %w", err)
	}

	// Convert to an internal type using the scheme, the conversion functions
	// must be registered in the scheme for this to work.
	err = e.scheme.Convert(versioned, IP(&internal), nil)
	if err != nil {
		return internal, fmt.Errorf("could not convert object to internal version: %w", err)
	}

	return internal, nil
}

// ObjectPtr is a type constraint. It is used to validate a pointer of a given
// type implements Object
type ObjectPtr[T any] interface {
	*T
	runtime.Object
}
