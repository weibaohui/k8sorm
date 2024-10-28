package kom

import (
	"context"
	"strings"

	"github.com/google/gnostic-models/openapiv2"
	"github.com/weibaohui/kom/kom/doc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/klog/v2"
)

type status struct {
	kubectl *Kubectl
}

func (s *status) APIResources() []*metav1.APIResource {
	cluster := s.kubectl.parentCluster()
	return cluster.apiResources
}
func (s *status) CRDList() []*unstructured.Unstructured {
	cluster := s.kubectl.parentCluster()
	return cluster.crdList
}
func (s *status) Docs() *doc.Docs {
	cluster := s.kubectl.parentCluster()
	return cluster.docs
}
func (s *status) ServerVersion() *version.Info {
	cluster := s.kubectl.parentCluster()
	return cluster.serverVersion
}

// 获取版本信息
func (k *Kubectl) initializeServerVersion() *version.Info {
	versionInfo, err := k.Client().Discovery().ServerVersion()
	if err != nil {
		klog.V(2).Infof("Error getting server version: %v\n", err)
	}
	return versionInfo
}

func (k *Kubectl) getOpenAPISchema() *openapi_v2.Document {
	openAPISchema, err := k.Client().Discovery().OpenAPISchema()
	if err != nil {
		klog.V(2).Infof("Error fetching OpenAPI schema: %v\n", err)
		return nil
	}
	return openAPISchema
}

func (k *Kubectl) initializeCRDList() []*unstructured.Unstructured {
	crdList, _ := k.listResources(context.TODO(), "CustomResourceDefinition", "")
	return crdList
}
func (k *Kubectl) initializeAPIResources() (apiResources []*metav1.APIResource) {
	// 提取ApiResources
	_, lists, _ := k.Client().Discovery().ServerGroupsAndResources()
	for _, list := range lists {
		resources := list.APIResources
		ver := list.GroupVersionKind().Version
		group := list.GroupVersionKind().Group
		groupVersion := list.GroupVersion
		gvs := strings.Split(groupVersion, "/")
		if len(gvs) == 2 {
			group = gvs[0]
			ver = gvs[1]
		} else {
			// 只有version的情况"v1"
			ver = groupVersion
		}

		for _, resource := range resources {
			resource.Group = group
			resource.Version = ver
			apiResources = append(apiResources, &resource)
		}
	}
	return apiResources
}
