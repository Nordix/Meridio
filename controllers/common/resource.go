package common

import (
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

/* Example data layout of a resource template file:

templates:
- name: small
  resources:
    limits:
      cpu: 100m
      memory: 32Mi
    requests:
      cpu: 30m
      memory: 16Mi
- name: medium
  resources:
    limits:
      cpu: 200m
      memory: 128Mi
    requests:
      cpu: 50m
      memory: 32Mi
- name: large
  resources:
    limits:
      cpu: "1"
      memory: 256Mi
    requests:
      cpu: 200m
      memory: 64Mi
- name: xlarge
  resources:
    limits:
      cpu: "4"
      memory: 512Mi
    requests:
      cpu: "2"
      memory: 128Mi
*/

type NamedResourceRequirements struct {
	Name      string                      `json:"name" protobuf:"bytes,1,opt,name=name"`
	Resources corev1.ResourceRequirements `json:"resources,omitempty" protobuf:"bytes,2,opt,name=resources"`
}

// ResourceRequirementTemplates -
// Describes the data layout of a resource requirement template file
type ResourceRequirementTemplates struct {
	Templates []NamedResourceRequirements `json:"templates,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,1,rep,name=templates"`
}

// getResourceRequirementTemplates -
// Reads resource requirement templates from file
func getResourceRequirementTemplates(f string) (*ResourceRequirementTemplates, error) {
	data, err := os.Open(f)
	if err != nil {
		return nil, fmt.Errorf("open %s error: %s", f, err)
	}
	t := &ResourceRequirementTemplates{}
	err = yaml.NewYAMLOrJSONDecoder(data, 4096).Decode(t)
	if err != nil {
		return nil, fmt.Errorf("decode %s error: %s", f, err)
	}
	return t, nil
}

// GetResourceRequirementAnnotation -
// Gets ResourceRequirementKey annotation based on param 'from'
func GetResourceRequirementAnnotation(from *metav1.ObjectMeta) (string, bool) {
	val, ok := from.Annotations[ResourceRequirementKey]
	return val, ok
}

// SetResourceRequirementAnnotation -
// Sets ResourceRequirementKey annotation based on param 'from'.
func SetResourceRequirementAnnotation(from *metav1.ObjectMeta, into *metav1.ObjectMeta) {
	if val, ok := GetResourceRequirementAnnotation(from); ok && val != "" {
		// annotatate param 'into' with ResourceRequirementKey
		if into.Annotations == nil {
			into.Annotations = make(map[string]string)
		}
		into.Annotations[ResourceRequirementKey] = val
	} else {
		// remove annotation from param 'into'
		if _, ok := GetResourceRequirementAnnotation(into); ok {
			delete(into.Annotations, ResourceRequirementKey)
		}
	}
}

// GetContainerResourceRequirements -
// Reads and searches template resource requirements for container.
// (A template resource requirement with param 'templateName' must exist for a match)
func GetContainerResourceRequirements(containerName, templateName string) (*corev1.ResourceRequirements, error) {
	rrt, err := getResourceRequirementTemplates(fmt.Sprintf("%s/%s", ResourceRequirementTemplatePath, containerName))
	if err != nil {
		return nil, err
	}
	for _, template := range rrt.Templates {
		if template.Name == templateName {
			return &template.Resources, nil
		}
	}
	return nil, fmt.Errorf("container %s, not found resource requirement template: %s", containerName, templateName)
}

// SetContainerResourceRequirements -
// Finds and sets resource requirements for container.
func SetContainerResourceRequirements(from *metav1.ObjectMeta, container *corev1.Container) error {
	if val, ok := GetResourceRequirementAnnotation(from); ok && val != "" {
		rr, err := GetContainerResourceRequirements(container.Name, val)
		if err != nil {
			return err
		}
		container.Resources = *rr
	}
	return nil
}
