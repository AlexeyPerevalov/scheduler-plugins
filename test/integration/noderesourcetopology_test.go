/*
Copyright 2021 The Kubernetes Authors.

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

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"

	"k8s.io/apiextensions-apiserver/test/integration/fixtures"
	"k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	clientset "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/controlplane"
	"k8s.io/kubernetes/pkg/scheduler"
	schedapi "k8s.io/kubernetes/pkg/scheduler/apis/config"
	fwkruntime "k8s.io/kubernetes/pkg/scheduler/framework/runtime"
	st "k8s.io/kubernetes/pkg/scheduler/testing"
	//"k8s.io/kubernetes/test/integration"
	//testapiserver "k8s.io/kubernetes/test/integration/apiserver"
	"k8s.io/kubernetes/test/integration/framework"
	testutils "k8s.io/kubernetes/test/integration/util"
	imageutils "k8s.io/kubernetes/test/utils/image"

	topologyv1alpha1 "github.com/k8stopologyawareschedwg/noderesourcetopology-api/pkg/apis/topology/v1alpha1"
	topologyclientset "github.com/k8stopologyawareschedwg/noderesourcetopology-api/pkg/generated/clientset/versioned"
	scheconfig "sigs.k8s.io/scheduler-plugins/pkg/apis/config"
	"sigs.k8s.io/scheduler-plugins/pkg/noderesourcetopology"
	"sigs.k8s.io/scheduler-plugins/test/util"
)

const (
	cpu    = string(v1.ResourceCPU)
	memory = string(v1.ResourceMemory)
)

func setup(t *testing.T, groupVersions ...schema.GroupVersion) (*httptest.Server, *clientset.Clientset, framework.CloseFunc) {
	return setupWithResources(t, groupVersions, nil)
}

func setupWithOptions(t *testing.T, opts *framework.MasterConfigOptions, groupVersions ...schema.GroupVersion) (*httptest.Server, *clientset.Clientset, framework.CloseFunc) {
	return setupWithResourcesWithOptions(t, opts, groupVersions, nil)
}

func setupWithResources(t *testing.T, groupVersions []schema.GroupVersion, resources []schema.GroupVersionResource) (*httptest.Server, *clientset.Clientset, framework.CloseFunc) {
	return setupWithResourcesWithOptions(t, &framework.MasterConfigOptions{}, groupVersions, resources)
}

func setupWithResourcesWithOptions(t *testing.T, opts *framework.MasterConfigOptions, groupVersions []schema.GroupVersion, resources []schema.GroupVersionResource) (*httptest.Server, *clientset.Clientset, framework.CloseFunc) {
	masterConfig := framework.NewIntegrationTestMasterConfigWithOptions(opts)
	if len(groupVersions) > 0 || len(resources) > 0 {
		resourceConfig := controlplane.DefaultAPIResourceConfigSource()
		resourceConfig.EnableVersions(groupVersions...)
		resourceConfig.EnableResources(resources...)
		masterConfig.ExtraConfig.APIResourceConfigSource = resourceConfig
	}
	masterConfig.GenericConfig.OpenAPIConfig = framework.DefaultOpenAPIConfig()
	_, s, closeFn := framework.RunAMaster(masterConfig)

	clientSet, err := clientset.NewForConfig(&restclient.Config{Host: s.URL, QPS: -1})
	if err != nil {
		t.Fatalf("Error in create clientset: %v", err)
	}
	return s, clientSet, closeFn
}

func TestTopologyMatchPlugin(t *testing.T) {
	todo := context.TODO()
	ctx, cancelFunc := context.WithCancel(todo)
	testCtx := &testutils.TestContext{
		Ctx:      ctx,
		CancelFn: cancelFunc,
		CloseFn:  func() {},
	}
	registry := fwkruntime.Registry{noderesourcetopology.Name: noderesourcetopology.New}
	t.Log("create apiserver")

	tearDown, config, srvOpts, err := fixtures.StartDefaultServer(t)
	if err != nil {
		t.Fatal(err)
	}

	optJson, err := json.Marshal(srvOpts)
	if err != nil {
		t.Log("failed to srvopts")
	}
	t.Logf("apiserver created: %s", string(optJson))
	defer tearDown()

	s, cs, closeFn := setup(t)
	defer closeFn()

	t.Log("apiExtensionClient")
	apiExtensionClient, err := apiextensionsclient.NewForConfig(config)

	//apiExtensionClient, err := apiextensionsclient.NewForConfig(&restclient.Config{Host: s.URL})

	if err != nil {
		t.Fatal(err)
	}

	t.Log("apiExtensionClient created")

	kubeConfigPath := util.BuildKubeConfigFile(&restclient.Config{Host: s.URL})
	if len(kubeConfigPath) == 0 {
		t.Fatal("Build KubeConfigFile failed")
	}
	defer os.RemoveAll(kubeConfigPath)

	t.Log("create crd")

	dynamicClient, err := dynamic.NewForConfig(config)

	if err != nil {
		t.Fatal(err)
	}

	crd := makeNodeResourceTopologyCRD()
	_, err = fixtures.CreateNewV1CustomResourceDefinition(crd, apiExtensionClient, dynamicClient)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("created crd")
	//if _, err := apiExtensionClient.ApiextensionsV1().CustomResourceDefinitions().Create(ctx, makeNodeResourceTopologyCRD(), metav1.CreateOptions{}); err != nil {
	//t.Fatal(err)
	//}

	//clientSet := clientset.NewForConfigOrDie(&restclient.Config{Host: s.URL})

	clientSet := clientset.NewForConfigOrDie(config)

	crdGVR := schema.GroupVersionResource{Group: crd.Spec.Group, Version: crd.Spec.Versions[0].Name, Resource: "noderesourcetopologies"}

	topologyClient, err := topologyclientset.NewForConfig(&restclient.Config{Host: s.URL})
	if err != nil {
		t.Fatal(err)
	}

	if err = wait.Poll(100*time.Millisecond, 3*time.Second, func() (done bool, err error) {
		//if crdList, err := apiExtensionClient.ApiextensionsV1().CustomResourceDefinitions().List(ctx, metav1.ListOptions{}); err == nil {
		//t.Logf("CRD: %v", crdList)
		//for _, item := range crdList.Items {
		//jsonItem, err := json.Marshal(item)
		//if err != nil {
		//t.Logf("Failed to parse: %v", err)
		//return false, err
		//}
		//t.Logf("Parsed json crd item: %s", string(jsonItem))
		//if item.Spec.Group == "topology.node.k8s.io" {
		//t.Logf("topology crd was successfully created!")
		//return true, nil
		//}
		//}
		//} else {
		//t.Logf("Can't request CRD: %v", err)
		//return false, err
		//}

		//groupList, _, err := clientSet.ServerGroupsAndResources()

		t.Log("Request group and resources")
		groupList, apiList, err := clientSet.ServerGroupsAndResources()
		jsonString, _ := json.Marshal(groupList)
		t.Logf("groupList: %s", string(jsonString))
		if err != nil {
			t.Logf("Can't request CRD: %v", err)
			return false, nil
		}
		apiJson, _ := json.Marshal(apiList)
		t.Logf("apiList: %s", string(apiJson))
		for _, group := range groupList {
			if group.Name == "topology.node.k8s.io" {
				t.Logf("topology crd was successfully created!")
				return true, nil
			}
		}
		t.Log("waiting for crd api ready")
		return false, nil
	}); err != nil {
		t.Fatalf("Waiting for crd read time out: %v", err)
	}

	t.Log("Create namespace")

	nameSpaceName := fmt.Sprintf("integration-test-%v", string(uuid.NewUUID()))

	ns, err := cs.CoreV1().Namespaces().Create(ctx, &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: nameSpaceName}}, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		t.Fatalf("Failed to create namespace integration test ns: %v", err)
	}

	crclient := dynamicClient.Resource(crdGVR).Namespace(nameSpaceName)
	t.Logf("crclient: %v", crclient)

	t.Log("namespace created")

	autoCreate := false
	t.Logf("namespace %+v", ns.Name)
	_, err = cs.CoreV1().ServiceAccounts(ns.Name).Create(ctx, &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: ns.Name}, AutomountServiceAccountToken: &autoCreate}, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		t.Fatalf("Failed to create ns default: %v", err)
	} else {
		t.Log("ns default created")
	}

	testCtx.NS = ns
	testCtx.ClientSet = cs

	args := &scheconfig.NodeResourceTopologyMatchArgs{
		KubeConfigPath: kubeConfigPath,
		Namespaces:     []string{ns.Name},
	}

	profile := schedapi.KubeSchedulerProfile{
		SchedulerName: v1.DefaultSchedulerName,
		Plugins: &schedapi.Plugins{
			Filter: &schedapi.PluginSet{
				Enabled: []schedapi.Plugin{
					{Name: noderesourcetopology.Name},
				},
			},
		},
		PluginConfig: []schedapi.PluginConfig{
			{
				Name: noderesourcetopology.Name,
				Args: args,
			},
		},
	}

	t.Log("Starting scheduler")
	testCtx = util.InitTestSchedulerWithOptions(
		t,
		testCtx,
		true,
		scheduler.WithProfiles(profile),
		scheduler.WithFrameworkOutOfTreeRegistry(registry),
	)
	t.Log("init scheduler success")
	defer testutils.CleanupTest(t, testCtx)

	// Create a Node.
	resList := map[v1.ResourceName]string{
		v1.ResourceCPU:    "4",
		v1.ResourceMemory: "100Gi",
		v1.ResourcePods:   "32",
	}
	for _, nodeName := range []string{"fake-node-1", "fake-node-2"} {
		newNode := st.MakeNode().Name(nodeName).Label("node", nodeName).Capacity(resList).Obj()
		n, err := cs.CoreV1().Nodes().Create(ctx, newNode, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("Failed to create Node %q: %v", nodeName, err)
		}

		t.Logf(" Node %s created: %v", nodeName, n)
	}

	nodeList, err := cs.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	t.Logf(" NodeList: %v", nodeList)
	pause := imageutils.GetPauseImageName()
	for _, tt := range []struct {
		name                   string
		pods                   []*v1.Pod
		nodeResourceTopologies []*topologyv1alpha1.NodeResourceTopology
		expectedNodes          []string
	}{
		{
			name: "Filtering out nodes that cannot fit resources on a single numa node in case of Guaranteed pod",
			pods: []*v1.Pod{
				withContainer(withReqAndLimit(st.MakePod().Namespace(ns.Name).Name("topology-aware-scheduler-pod"), map[v1.ResourceName]string{v1.ResourceCPU: "4", v1.ResourceMemory: "50Gi"}).Obj(), pause),
			},
			nodeResourceTopologies: []*topologyv1alpha1.NodeResourceTopology{
				{
					ObjectMeta:       metav1.ObjectMeta{Name: "fake-node-1", Namespace: ns.Name},
					TopologyPolicies: []string{string(topologyv1alpha1.SingleNUMANodeContainerLevel)},
					Zones: topologyv1alpha1.ZoneList{
						topologyv1alpha1.Zone{
							Name: "node-0",
							Type: "Node",
							Resources: topologyv1alpha1.ResourceInfoList{
								topologyv1alpha1.ResourceInfo{
									Name:        cpu,
									Allocatable: intstr.FromString("2"),
									Capacity:    intstr.FromString("2"),
								},
								topologyv1alpha1.ResourceInfo{
									Name:        memory,
									Allocatable: intstr.Parse("8Gi"),
									Capacity:    intstr.Parse("8Gi"),
								},
							},
						},
						topologyv1alpha1.Zone{
							Name: "node-1",
							Type: "Node",
							Resources: topologyv1alpha1.ResourceInfoList{
								topologyv1alpha1.ResourceInfo{
									Name:        cpu,
									Allocatable: intstr.FromString("2"),
									Capacity:    intstr.FromString("2"),
								},
								topologyv1alpha1.ResourceInfo{
									Name:        memory,
									Allocatable: intstr.Parse("8Gi"),
									Capacity:    intstr.Parse("8Gi"),
								},
							},
						},
					},
				},
				{
					ObjectMeta:       metav1.ObjectMeta{Name: "fake-node-2", Namespace: ns.Name},
					TopologyPolicies: []string{string(topologyv1alpha1.SingleNUMANodeContainerLevel)},
					Zones: topologyv1alpha1.ZoneList{
						topologyv1alpha1.Zone{
							Name: "node-0",
							Type: "Node",
							Resources: topologyv1alpha1.ResourceInfoList{
								topologyv1alpha1.ResourceInfo{
									Name:        cpu,
									Allocatable: intstr.FromString("4"),
									Capacity:    intstr.FromString("4"),
								},
								topologyv1alpha1.ResourceInfo{
									Name:        memory,
									Allocatable: intstr.Parse("8Gi"),
									Capacity:    intstr.Parse("8Gi"),
								},
							},
						},
						topologyv1alpha1.Zone{
							Name: "node-1",
							Type: "Node",
							Resources: topologyv1alpha1.ResourceInfoList{
								topologyv1alpha1.ResourceInfo{
									Name:        cpu,
									Allocatable: intstr.FromString("0"),
									Capacity:    intstr.FromString("0"),
								},
								topologyv1alpha1.ResourceInfo{
									Name:        memory,
									Allocatable: intstr.Parse("8Gi"),
									Capacity:    intstr.Parse("8Gi"),
								},
							},
						},
					},
				},
			},
			expectedNodes: []string{"fake-node-2"},
		},
		{
			name: "Scheduling of a burstable pod requesting only cpus",
			pods: []*v1.Pod{
				withContainer(st.MakePod().Namespace(ns.Name).Name("topology-aware-scheduler-pod").Req(map[v1.ResourceName]string{v1.ResourceCPU: "4"}).Obj(), pause),
			},
			nodeResourceTopologies: []*topologyv1alpha1.NodeResourceTopology{
				{
					ObjectMeta:       metav1.ObjectMeta{Name: "fake-node-1", Namespace: ns.Name},
					TopologyPolicies: []string{string(topologyv1alpha1.SingleNUMANodeContainerLevel)},
					Zones: topologyv1alpha1.ZoneList{
						topologyv1alpha1.Zone{
							Name: "node-0",
							Type: "Node",
							Resources: topologyv1alpha1.ResourceInfoList{
								topologyv1alpha1.ResourceInfo{
									Name:        cpu,
									Allocatable: intstr.FromString("4"),
									Capacity:    intstr.FromString("4"),
								},
							},
						},
						topologyv1alpha1.Zone{
							Name: "node-1",
							Type: "Node",
							Resources: topologyv1alpha1.ResourceInfoList{
								topologyv1alpha1.ResourceInfo{
									Name:        cpu,
									Allocatable: intstr.FromString("0"),
									Capacity:    intstr.FromString("0"),
								},
							},
						},
					},
				},
				{
					ObjectMeta:       metav1.ObjectMeta{Name: "fake-node-2", Namespace: ns.Name},
					TopologyPolicies: []string{string(topologyv1alpha1.SingleNUMANodeContainerLevel)},
					Zones: topologyv1alpha1.ZoneList{
						topologyv1alpha1.Zone{
							Name: "node-0",
							Type: "Node",
							Resources: topologyv1alpha1.ResourceInfoList{
								topologyv1alpha1.ResourceInfo{
									Name:        cpu,
									Allocatable: intstr.FromString("2"),
									Capacity:    intstr.FromString("2"),
								},
							},
						},
						topologyv1alpha1.Zone{
							Name: "node-1",
							Type: "Node",
							Resources: topologyv1alpha1.ResourceInfoList{
								topologyv1alpha1.ResourceInfo{
									Name:        cpu,
									Allocatable: intstr.FromString("2"),
									Capacity:    intstr.FromString("2"),
								},
							},
						},
					},
				},
			},
			expectedNodes: []string{"fake-node-1", "fake-node-2"},
		},
		{
			name: "Scheduling of a burstable pod requesting only memory",
			pods: []*v1.Pod{
				withContainer(st.MakePod().Namespace(ns.Name).Name("topology-aware-scheduler-pod").Req(map[v1.ResourceName]string{v1.ResourceMemory: "5Gi"}).Obj(), pause),
			},
			nodeResourceTopologies: []*topologyv1alpha1.NodeResourceTopology{
				{
					ObjectMeta:       metav1.ObjectMeta{Name: "fake-node-1", Namespace: ns.Name},
					TopologyPolicies: []string{string(topologyv1alpha1.SingleNUMANodeContainerLevel)},
					Zones: topologyv1alpha1.ZoneList{
						topologyv1alpha1.Zone{
							Name: "node-0",
							Type: "Node",
							Resources: topologyv1alpha1.ResourceInfoList{
								topologyv1alpha1.ResourceInfo{
									Name:        "foo",
									Allocatable: intstr.FromString("2"),
									Capacity:    intstr.FromString("2"),
								},
							},
						},
						topologyv1alpha1.Zone{
							Name: "node-1",
							Type: "Node",
							Resources: topologyv1alpha1.ResourceInfoList{
								topologyv1alpha1.ResourceInfo{
									Name:        "foo",
									Allocatable: intstr.FromString("2"),
									Capacity:    intstr.FromString("2"),
								},
							},
						},
					},
				},
				{
					ObjectMeta:       metav1.ObjectMeta{Name: "fake-node-2", Namespace: ns.Name},
					TopologyPolicies: []string{"foo"},
					Zones: topologyv1alpha1.ZoneList{
						topologyv1alpha1.Zone{
							Name: "node-0",
							Type: "Node",
							Resources: topologyv1alpha1.ResourceInfoList{
								topologyv1alpha1.ResourceInfo{
									Name:        "foo",
									Allocatable: intstr.FromString("2"),
									Capacity:    intstr.FromString("2"),
								},
							},
						},
						topologyv1alpha1.Zone{
							Name: "node-1",
							Type: "Node",
							Resources: topologyv1alpha1.ResourceInfoList{
								topologyv1alpha1.ResourceInfo{
									Name:        "foo",
									Allocatable: intstr.FromString("2"),
									Capacity:    intstr.FromString("2"),
								},
							},
						},
					},
				},
			},
			expectedNodes: []string{"fake-node-1", "fake-node-2"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Start-topology-match-test %v", tt.name)

			defer cleanupNodeResourceTopologies(ctx, topologyClient, tt.nodeResourceTopologies)

			if err := createNodeResourceTopologies(ctx, topologyClient, tt.nodeResourceTopologies); err != nil {
				t.Fatal(err)
			}

			defer testutils.CleanupPods(cs, t, tt.pods)
			// Create Pods
			for _, p := range tt.pods {
				t.Logf("Creating Pod %q", p.Name)
				_, err := cs.CoreV1().Pods(ns.Name).Create(context.TODO(), p, metav1.CreateOptions{})
				if err != nil {
					t.Fatalf("Failed to create Pod %q: %v", p.Name, err)
				}
				t.Logf("pod created: %v", p)
			}

			for _, p := range tt.pods {
				// Wait for the pod to be scheduled.
				err = wait.Poll(1*time.Second, 20*time.Second, func() (bool, error) {
					return podScheduled(cs, ns.Name, p.Name), nil
				})
				if err != nil {
					t.Errorf("pod %q to be scheduled, error: %v", p.Name, err)
				}

				t.Logf("p scheduled: %v", p.Name)

				// The other pods should be scheduled on the small nodes.
				nodeName, err := getNodeName(cs, ns.Name, p.Name)
				if err != nil {
					t.Log(err)
				}
				if contains(tt.expectedNodes, nodeName) {
					t.Logf("Pod %q is on a nodes as expected.", p.Name)
				} else {
					t.Errorf("Pod %s is expected on node %s, but found on node %s",
						p.Name, tt.expectedNodes, nodeName)
				}

			}
			t.Logf("case %v finished", tt.name)
		})
	}
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// getNodeName returns the name of the node if a node has assigned to the given pod
func getNodeName(c clientset.Interface, podNamespace, podName string) (string, error) {
	pod, err := c.CoreV1().Pods(podNamespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return pod.Spec.NodeName, nil
}

func makeNodeResourceTopologyCRD() *apiextensionsv1.CustomResourceDefinition {
	return &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "noderesourcetopologies.topology.node.k8s.io",
			Annotations: map[string]string{
				"api-approved.kubernetes.io": "https://github.com/kubernetes/enhancements/pull/1870",
			},
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "topology.node.k8s.io",
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Plural:   "noderesourcetopologies",
				Singular: "noderesourcetopology",
				ShortNames: []string{
					"node-res-topo",
				},
				Kind: "NodeResourceTopology",
			},
			Scope: "Namespaced",
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{
					Name: "v1alpha1",
					Schema: &apiextensionsv1.CustomResourceValidation{
						OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
							Type: "object",
							Properties: map[string]apiextensionsv1.JSONSchemaProps{
								"topologyPolicies": {
									Type: "array",
									Items: &apiextensionsv1.JSONSchemaPropsOrArray{
										Schema: &apiextensionsv1.JSONSchemaProps{
											Type: "string",
										},
									},
								},
								"zones": {
									Type: "array",
									Items: &apiextensionsv1.JSONSchemaPropsOrArray{
										Schema: &apiextensionsv1.JSONSchemaProps{
											Type: "object",
											Properties: map[string]apiextensionsv1.JSONSchemaProps{
												"name": {
													Type: "string",
												},
												"type": {
													Type: "string",
												},
												"parent": {
													Type: "string",
												},
												"resources": {
													Type: "array",
													Items: &apiextensionsv1.JSONSchemaPropsOrArray{
														Schema: &apiextensionsv1.JSONSchemaProps{
															Type: "object",
															Properties: map[string]apiextensionsv1.JSONSchemaProps{
																"name": {
																	Type: "string",
																},
																"capacity": {
																	XIntOrString: true,
																},
																"allocatable": {
																	XIntOrString: true,
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
					Served:  true,
					Storage: true,
				},
			},
		},
	}
}

func createNodeResourceTopologies(ctx context.Context, topologyClient *topologyclientset.Clientset, noderesourcetopologies []*topologyv1alpha1.NodeResourceTopology) error {
	for _, nrt := range noderesourcetopologies {
		_, err := topologyClient.TopologyV1alpha1().NodeResourceTopologies(nrt.Namespace).Create(ctx, nrt, metav1.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			return err
		}
	}
	return nil
}

func cleanupNodeResourceTopologies(ctx context.Context, topologyClient *topologyclientset.Clientset, noderesourcetopologies []*topologyv1alpha1.NodeResourceTopology) {
	for _, nrt := range noderesourcetopologies {
		err := topologyClient.TopologyV1alpha1().NodeResourceTopologies(nrt.Namespace).Delete(ctx, nrt.Name, metav1.DeleteOptions{})
		if err != nil {
			klog.Errorf("clean up NodeResourceTopologies (%v/%v) error %s", nrt.Namespace, nrt.Name, err.Error())
		}
	}
}

func withContainer(pod *v1.Pod, image string) *v1.Pod {
	pod.Spec.Containers[0].Name = "con0"
	pod.Spec.Containers[0].Image = image
	return pod
}

// withReqAndLimit adds a new container to the inner pod with given resource map.
func withReqAndLimit(p *st.PodWrapper, resMap map[v1.ResourceName]string) *st.PodWrapper {
	if len(resMap) == 0 {
		return p
	}
	res := v1.ResourceList{}
	for k, v := range resMap {
		res[k] = resource.MustParse(v)
	}
	p.Spec.Containers = append(p.Spec.Containers, v1.Container{
		Resources: v1.ResourceRequirements{
			Requests: res,
			Limits:   res,
		},
	})
	return p
}
