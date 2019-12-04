package toolchain

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CreateFromYAML creates a resource from a YAML manifest
func CreateFromYAML(s *runtime.Scheme, cl client.Client, content []byte) error {
	decoder := serializer.NewCodecFactory(s).UniversalDeserializer()
	obj := unstructured.Unstructured{}
	_, _, err := decoder.Decode(content, nil, &obj)
	if err != nil {
		return err
	}
	if err = cl.Create(context.TODO(), &obj); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
} 
