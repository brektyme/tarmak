// +build !ignore_autogenerated

/*
Copyright 2017 The Kubernetes Authors.

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

// This file was autogenerated by defaulter-gen. Do not edit it manually!

package v1alpha1

import (
	cluster_v1alpha1 "github.com/jetstack/tarmak/pkg/apis/cluster/v1alpha1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// RegisterDefaults adds defaulters functions to the given scheme.
// Public to allow building arbitrary schemes.
// All generated defaulters are covering - they call all nested defaulters.
func RegisterDefaults(scheme *runtime.Scheme) error {
	scheme.AddTypeDefaultingFunc(&Config{}, func(obj interface{}) { SetObjectDefaults_Config(obj.(*Config)) })
	scheme.AddTypeDefaultingFunc(&ConfigList{}, func(obj interface{}) { SetObjectDefaults_ConfigList(obj.(*ConfigList)) })
	scheme.AddTypeDefaultingFunc(&Provider{}, func(obj interface{}) { SetObjectDefaults_Provider(obj.(*Provider)) })
	scheme.AddTypeDefaultingFunc(&ProviderList{}, func(obj interface{}) { SetObjectDefaults_ProviderList(obj.(*ProviderList)) })
	return nil
}

func SetObjectDefaults_Config(in *Config) {
	SetDefaults_Config(in)
	for i := range in.Clusters {
		a := &in.Clusters[i]
		cluster_v1alpha1.SetDefaults_Cluster(a)
		for j := range a.ServerPools {
			b := &a.ServerPools[j]
			cluster_v1alpha1.SetDefaults_ServerPool(b)
			for k := range b.Volumes {
				c := &b.Volumes[k]
				cluster_v1alpha1.SetDefaults_Volume(c)
			}
		}
	}
	for i := range in.Providers {
		a := &in.Providers[i]
		SetObjectDefaults_Provider(a)
	}
}

func SetObjectDefaults_ConfigList(in *ConfigList) {
	for i := range in.Items {
		a := &in.Items[i]
		SetObjectDefaults_Config(a)
	}
}

func SetObjectDefaults_Provider(in *Provider) {
	SetDefaults_Provider(in)
}

func SetObjectDefaults_ProviderList(in *ProviderList) {
	for i := range in.Items {
		a := &in.Items[i]
		SetObjectDefaults_Provider(a)
	}
}