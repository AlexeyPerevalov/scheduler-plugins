// +build !ignore_autogenerated

/*
Copyright The Kubernetes Authors.

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

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1beta1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
	v1 "k8s.io/kube-scheduler/config/v1"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CapacitySchedulingArgs) DeepCopyInto(out *CapacitySchedulingArgs) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	if in.KubeConfigPath != nil {
		in, out := &in.KubeConfigPath, &out.KubeConfigPath
		*out = new(string)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CapacitySchedulingArgs.
func (in *CapacitySchedulingArgs) DeepCopy() *CapacitySchedulingArgs {
	if in == nil {
		return nil
	}
	out := new(CapacitySchedulingArgs)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CapacitySchedulingArgs) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CoschedulingArgs) DeepCopyInto(out *CoschedulingArgs) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	if in.PermitWaitingTimeSeconds != nil {
		in, out := &in.PermitWaitingTimeSeconds, &out.PermitWaitingTimeSeconds
		*out = new(int64)
		**out = **in
	}
	if in.DeniedPGExpirationTimeSeconds != nil {
		in, out := &in.DeniedPGExpirationTimeSeconds, &out.DeniedPGExpirationTimeSeconds
		*out = new(int64)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CoschedulingArgs.
func (in *CoschedulingArgs) DeepCopy() *CoschedulingArgs {
	if in == nil {
		return nil
	}
	out := new(CoschedulingArgs)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CoschedulingArgs) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodeResourceTopologyMatchArgs) DeepCopyInto(out *NodeResourceTopologyMatchArgs) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	if in.KubeConfig != nil {
		in, out := &in.KubeConfig, &out.KubeConfig
		*out = new(string)
		**out = **in
	}
	if in.MasterOverride != nil {
		in, out := &in.MasterOverride, &out.MasterOverride
		*out = new(string)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodeResourceTopologyMatchArgs.
func (in *NodeResourceTopologyMatchArgs) DeepCopy() *NodeResourceTopologyMatchArgs {
	if in == nil {
		return nil
	}
	out := new(NodeResourceTopologyMatchArgs)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NodeResourceTopologyMatchArgs) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodeResourcesAllocatableArgs) DeepCopyInto(out *NodeResourcesAllocatableArgs) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	if in.Resources != nil {
		in, out := &in.Resources, &out.Resources
		*out = make([]v1.ResourceSpec, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodeResourcesAllocatableArgs.
func (in *NodeResourcesAllocatableArgs) DeepCopy() *NodeResourcesAllocatableArgs {
	if in == nil {
		return nil
	}
	out := new(NodeResourcesAllocatableArgs)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NodeResourcesAllocatableArgs) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}
