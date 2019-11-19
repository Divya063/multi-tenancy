/*
Copyright 2019 The Kubernetes Authors.

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

package conversion

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
	v1helper "k8s.io/kubernetes/pkg/apis/core/v1/helper"
	"k8s.io/kubernetes/pkg/kubelet/envvars"

	"github.com/kubernetes-sigs/multi-tenancy/incubator/virtualcluster/pkg/apis/tenancy/v1alpha1"
	"github.com/kubernetes-sigs/multi-tenancy/incubator/virtualcluster/pkg/syncer/constants"
)

const (
	masterServiceNamespace = metav1.NamespaceDefault
)

var masterServices = sets.NewString("kubernetes")

// ToClusterKey make a unique id for a virtual cluster object.
// The key uses the format <namespace>-<name> unless <namespace> is empty, then
// it's just <name>.
func ToClusterKey(vc *v1alpha1.Virtualcluster) string {
	if len(vc.GetNamespace()) > 0 {
		return vc.GetNamespace() + "-" + vc.GetName()
	}
	return vc.GetName()
}

func ToSuperMasterNamespace(cluster, ns string) string {
	targetNamespace := strings.Join([]string{cluster, ns}, "-")
	if len(targetNamespace) > validation.DNS1123SubdomainMaxLength {
		digest := sha256.Sum256([]byte(targetNamespace))
		return targetNamespace[0:57] + "-" + hex.EncodeToString(digest[0:])[0:5]
	}
	return targetNamespace
}

func GetOwner(obj runtime.Object) (cluster, namespace string) {
	meta, err := meta.Accessor(obj)
	if err != nil {
		return "", ""
	}

	cluster = meta.GetAnnotations()[constants.LabelCluster]
	namespace = strings.TrimPrefix(meta.GetNamespace(), cluster+"-")
	return cluster, namespace
}

func BuildMetadata(cluster, targetNamespace string, obj runtime.Object) (runtime.Object, error) {
	target := obj.DeepCopyObject()
	m, err := meta.Accessor(target)
	if err != nil {
		return nil, err
	}

	uid := m.GetUID()

	resetMetadata(m)
	if len(targetNamespace) > 0 {
		m.SetNamespace(targetNamespace)
	}

	anno := m.GetAnnotations()
	if anno == nil {
		anno = map[string]string{}
	}
	anno[constants.LabelCluster] = cluster
	anno[constants.LabelUID] = string(uid)
	m.SetAnnotations(anno)

	return target, nil
}

func BuildSuperMasterNamespace(cluster string, obj runtime.Object) (runtime.Object, error) {
	target := obj.DeepCopyObject()
	m, err := meta.Accessor(target)
	if err != nil {
		return nil, err
	}

	resetMetadata(m)

	targetName := strings.Join([]string{cluster, m.GetName()}, "-")
	m.SetName(targetName)
	return target, nil
}

func resetMetadata(obj metav1.Object) {
	obj.SetGenerateName("")
	obj.SetSelfLink("")
	obj.SetUID("")
	obj.SetResourceVersion("")
	obj.SetGeneration(0)
	obj.SetCreationTimestamp(metav1.Time{})
	obj.SetDeletionTimestamp(nil)
	obj.SetDeletionGracePeriodSeconds(nil)
	obj.SetOwnerReferences(nil)
	obj.SetFinalizers(nil)
	obj.SetClusterName("")
	obj.SetInitializers(nil)
}

// MutatePod convert the meta data of containers to super master namespace.
// replace the service account token volume mounts to super master side one.
func MutatePod(namespace string, pod *corev1.Pod, vSASecret, SASecret *v1.Secret, services []v1.Service) {
	pod.Status = corev1.PodStatus{}
	pod.Spec.NodeName = ""

	// setup env var map
	serviceEnv := getServiceEnvVarMap(pod.Namespace, *pod.Spec.EnableServiceLinks, services)

	for i := range pod.Spec.Containers {
		// Inject env var from service
		// 1. Do nothing if it conflicts with user-defined one.
		// 2. Add remaining service environment vars
		envNameMap := make(map[string]struct{})
		for j, env := range pod.Spec.Containers[i].Env {
			if env.ValueFrom != nil && env.ValueFrom.FieldRef != nil && env.ValueFrom.FieldRef.FieldPath == "metadata.name" {
				pod.Spec.Containers[i].Env[j].ValueFrom = nil
				pod.Spec.Containers[i].Env[j].Value = pod.Name
			}
			if env.ValueFrom != nil && env.ValueFrom.FieldRef != nil && env.ValueFrom.FieldRef.FieldPath == "metadata.namespace" {
				pod.Spec.Containers[i].Env[j].ValueFrom = nil
				pod.Spec.Containers[i].Env[j].Value = namespace
			}
			envNameMap[env.Name] = struct{}{}
		}
		for k, v := range serviceEnv {
			if _, exists := envNameMap[k]; !exists {
				pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, v1.EnvVar{Name: k, Value: v})
			}
		}

		for j, volumeMount := range pod.Spec.Containers[i].VolumeMounts {
			if volumeMount.Name == vSASecret.Name {
				pod.Spec.Containers[i].VolumeMounts[j].Name = SASecret.Name
			}
		}
	}

	for i, volume := range pod.Spec.Volumes {
		if volume.Name == vSASecret.Name {
			pod.Spec.Volumes[i].Name = SASecret.Name
			pod.Spec.Volumes[i].Secret.SecretName = SASecret.Name
		}
	}
}

func getServiceEnvVarMap(ns string, enableServiceLinks bool, services []v1.Service) map[string]string {
	var (
		serviceMap = make(map[string]*v1.Service)
		m          = make(map[string]string)
	)

	// project the services in namespace ns onto the master services
	for i := range services {
		service := services[i]
		// ignore services where ClusterIP is "None" or empty
		if !v1helper.IsServiceIPSet(&service) {
			continue
		}
		serviceName := service.Name

		// We always want to add environment variabled for master services
		// from the master service namespace, even if enableServiceLinks is false.
		// We also add environment variables for other services in the same
		// namespace, if enableServiceLinks is true.
		if service.Namespace == masterServiceNamespace && masterServices.Has(serviceName) {
			if _, exists := serviceMap[serviceName]; !exists {
				serviceMap[serviceName] = &service
			}
		} else if service.Namespace == ns && enableServiceLinks {
			serviceMap[serviceName] = &service
		}
	}

	var mappedServices []*v1.Service
	for key := range serviceMap {
		mappedServices = append(mappedServices, serviceMap[key])
	}

	for _, e := range envvars.FromServices(mappedServices) {
		m[e.Name] = e.Value
	}
	return m
}

func MutateService(newService *corev1.Service) {
	newService.Spec.ClusterIP = ""
	for i := range newService.Spec.Ports {
		newService.Spec.Ports[i].NodePort = 0
	}
}
