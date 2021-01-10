/*
    Copyright (C) 2020 Accurics, Inc.

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

package k8sv1

import (
	"reflect"
	"testing"

	"github.com/accurics/terrascan/pkg/iac-providers/output"
	"github.com/accurics/terrascan/pkg/utils"
)

var (
	testJSONData = []byte(`{
		"kind": "Pod",
		"apiVersion": "v1",
		"metadata": {
		  "name": "simple"
		},
		"spec": {
		  "containers": [
			{
			  "name": "healthz",
			  "image": "k8s.gcr.io/exechealthz-amd64:1.2",
			  "args": [
				"-cmd=nslookup localhost"
			  ],
			  "ports": [
				{
				  "containerPort": 8080,
				  "protocol": "TCP"
				}
			  ]
			}
		  ]
		}
	  }`)

	testYAMLData = []byte(`apiVersion: v1
kind: Pod
metadata:
  name: myapp-pod
  annotations:
    terrascanSkip:
      - rule: accurics.kubernetes.IAM.109
        comment: reason to skip the rule
spec:
  containers:
    - name: myapp-container
      image: busybox`)

	testYAMLDataWithGenerateName = []byte(`apiVersion: v1
kind: CRD
metadata:
  generateName: myapp-pod-prefix-
  annotations:
    terrascanSkip: [accurics.kubernetes.IAM.109]
spec:
  containers:
    - name: myapp-container
      image: busybox`)
)

func TestK8sV1ExtractResource(t *testing.T) {
	type args struct {
		doc *utils.IacDocument
	}
	tests := []struct {
		name    string
		k       *K8sV1
		args    args
		want    *k8sResource
		wantErr bool
	}{
		{
			name: "empty document object",
			args: args{
				doc: &utils.IacDocument{},
			},
			wantErr: true,
		},
		{
			name: "json document object",
			args: args{
				doc: &utils.IacDocument{
					Type: "json",
					Data: testJSONData,
				},
			},
			want: &k8sResource{
				APIVersion: "v1",
				Kind:       "Pod",
				Metadata: k8sMetadata{
					Name: "simple",
				},
			},
		},
		{
			name: "yaml document object",
			args: args{
				doc: &utils.IacDocument{
					Type: "yaml",
					Data: testYAMLData,
				},
			},
			want: &k8sResource{
				APIVersion: "v1",
				Kind:       "Pod",
				Metadata: k8sMetadata{
					Name: "myapp-pod",
					Annotations: map[string]interface{}{
						terrascanSkip: []interface{}{map[string]interface{}{
							"rule":    "accurics.kubernetes.IAM.109",
							"comment": "reason to skip the rule",
						}},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &K8sV1{}
			got, _, err := k.extractResource(tt.args.doc)
			if (err != nil) != tt.wantErr {
				t.Errorf("K8sV1.extractResource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("K8sV1.extractResource() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestK8sV1GetNormalizedName(t *testing.T) {
	type args struct {
		kind string
	}
	tests := []struct {
		name string
		k    *K8sV1
		args args
		want string
	}{
		{
			name: "normalized name for pod",
			args: args{
				kind: "pod",
			},
			want: "kubernetes_pod",
		},
		{
			name: "normalized name for DaemonSet",
			args: args{
				kind: "DaemonSet",
			},
			want: "kubernetes_daemonset",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &K8sV1{}
			if got := k.getNormalizedName(tt.args.kind); got != tt.want {
				t.Errorf("K8sV1.getNormalizedName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestK8sV1Normalize(t *testing.T) {
	testRule := "accurics.kubernetes.IAM.109"
	testComment := "reason to skip the rule"

	testSkipRule := output.SkipRule{
		Rule:    testRule,
		Comment: testComment,
	}

	type args struct {
		doc *utils.IacDocument
	}
	tests := []struct {
		name    string
		k       *K8sV1
		args    args
		want    *output.ResourceConfig
		wantErr bool
	}{
		{
			name: "empty iac document object",
			args: args{
				&utils.IacDocument{},
			},
			wantErr: true,
		},
		{
			name: "valid iac document object",
			args: args{
				&utils.IacDocument{
					Type: "yaml",
					Data: testYAMLData,
				},
			},
			want: &output.ResourceConfig{
				ID:   "kubernetes_pod.myapp-pod.default",
				Name: "myapp-pod",
				Line: 0,
				Type: "kubernetes_pod",
				Config: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Pod",
					"metadata": map[string]interface{}{
						"annotations": map[string]interface{}{
							terrascanSkip: []interface{}{map[string]interface{}{
								"rule":    testRule,
								"comment": testComment,
							}},
						},
						"name": "myapp-pod",
					},
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"image": "busybox",
								"name":  "myapp-container",
							},
						},
					},
				},
				SkipRules: []string{testRule},
			},
		},
		{
			name: "valid iac document object with generateName",
			args: args{
				&utils.IacDocument{
					Type: "yaml",
					Data: testYAMLDataWithGenerateName,
				},
			},
			want: &output.ResourceConfig{
				ID:   "kubernetes_crd.myapp-pod-prefix-.default",
				Name: "myapp-pod-prefix-",
				Line: 0,
				Type: "kubernetes_crd",
				Config: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "CRD",
					"metadata": map[string]interface{}{
						"annotations": map[string]interface{}{
							terrascanSkip: []interface{}{testRule},
						},
						"generateName": "myapp-pod-prefix-",
					},
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"image": "busybox",
								"name":  "myapp-container",
							},
						},
					},
				},
				SkipRules: []output.SkipRule{testSkipRule},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &K8sV1{}
			got, err := k.Normalize(tt.args.doc)
			if (err != nil) != tt.wantErr {
				t.Errorf("K8sV1.Normalize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("K8sV1.Normalize() got = %+v, want = %+v", got, tt.want)
			}
		})
	}
}

func TestReadSkipRulesFromAnnotations(t *testing.T) {
	// test data
	testRuleA := "RuleA"
	testCommentA := "RuleA can be skipped"
	testRuleB := "RuleB"
	testCommentB := "RuleB must be skipped"
	testRuleC := "RuleC"
	testCommentC := "RuleC skipped"

	testSkipRule := output.SkipRule{Rule: testRuleA}

	type args struct {
		annotations map[string]interface{}
		resourceID  string
	}
	tests := []struct {
		name string
		args args
		want []output.SkipRule
	}{
		{
			name: "nil annotations",
			args: args{
				annotations: nil,
			},
		},
		{
			name: "annotations with no terrascanSkipRules",
			args: args{
				annotations: map[string]interface{}{
					"test": "test",
				},
			},
		},
		{
			name: "annotations with invalid terrascanSkipRules type",
			args: args{
				annotations: map[string]interface{}{
					terrascanSkip: "test",
				},
			},
			want: []output.SkipRule{},
		},
		{
			name: "annotations with invalid SkipRule object",
			args: args{
				annotations: map[string]interface{}{
					terrascanSkip: []interface{}{1},
				},
			},
			want: []output.SkipRule{},
		},
		{
			name: "annotations with invalid terrascanSkipRules rule value",
			args: args{
				annotations: map[string]interface{}{
					terrascanSkip: []interface{}{map[string]interface{}{
						terrascanSkipRule: 1,
					}},
				},
			},
			want: []output.SkipRule{},
		},
		{
			name: "annotations with one terrascanSkipRules",
			args: args{
				annotations: map[string]interface{}{
					terrascanSkip: []interface{}{map[string]interface{}{
						terrascanSkipRule: testRuleA,
					}},
				},
			},
			want: []output.SkipRule{
				{
					Rule: testRuleA,
				},
			},
		},
		{
			name: "annotations with multiple terrascanSkipRules",
			args: args{
				annotations: map[string]interface{}{
					terrascanSkip: []interface{}{
						map[string]interface{}{
							terrascanSkipRule:    testRuleA,
							terrascanSkipComment: testCommentA,
						},
						map[string]interface{}{
							terrascanSkipRule:    testRuleB,
							terrascanSkipComment: testCommentB,
						},
						map[string]interface{}{
							terrascanSkipRule:    testRuleC,
							terrascanSkipComment: testCommentC,
						}},
				},
			},
			want: []output.SkipRule{
				{
					Rule:    testRuleA,
					Comment: testCommentA,
				},
				{
					Rule:    testRuleB,
					Comment: testCommentB,
				},
				{
					Rule:    testRuleC,
					Comment: testCommentC,
				},
			},
		},
		{
			name: "annotations with invalid rule key in terrascanSkipRules",
			args: args{
				annotations: map[string]interface{}{
					terrascanSkip: []interface{}{
						map[string]interface{}{
							"skip-rule":          testRuleA,
							terrascanSkipComment: testCommentA,
						}},
				},
			},
			want: []output.SkipRule{},
		},
		{
			name: "annotations with no comment key in terrascanSkipRules",
			args: args{
				annotations: map[string]interface{}{
					terrascanSkip: []interface{}{
						map[string]interface{}{
							terrascanSkipRule: testRuleA,
						}},
				},
			},
			want: []output.SkipRule{testSkipRule},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := readSkipRulesFromAnnotations(tt.args.annotations, tt.args.resourceID); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("readSkipRulesFromAnnotations() = got %v, want %v", got, tt.want)
			}
		})
	}
}
