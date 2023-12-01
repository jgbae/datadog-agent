// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build kubeapiserver

package mutate

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic/fake"
)

func TestInjectAutoInstruConfig(t *testing.T) {
	tests := []struct {
		name           string
		pod            *corev1.Pod
		libsToInject   []libInfo
		expectedEnvKey string
		expectedEnvVal string
		wantErr        bool
	}{
		{
			name: "nominal case: java",
			pod:  fakePod("java-pod"),
			libsToInject: []libInfo{
				{
					lang:  "java",
					image: "gcr.io/datadoghq/dd-lib-java-init:v1",
				},
			},
			expectedEnvKey: "JAVA_TOOL_OPTIONS",
			expectedEnvVal: " -javaagent:/datadog-lib/dd-java-agent.jar -XX:OnError=/datadog-lib/continuousprofiler/tmp/dd_crash_uploader.sh -XX:ErrorFile=/datadog-lib/continuousprofiler/tmp/hs_err_pid_%p.log",
			wantErr:        false,
		},
		{
			name: "JAVA_TOOL_OPTIONS not empty",
			pod:  fakePodWithEnvValue("java-pod", "JAVA_TOOL_OPTIONS", "predefined"),
			libsToInject: []libInfo{
				{
					lang:  "java",
					image: "gcr.io/datadoghq/dd-lib-java-init:v1",
				},
			},
			expectedEnvKey: "JAVA_TOOL_OPTIONS",
			expectedEnvVal: "predefined -javaagent:/datadog-lib/dd-java-agent.jar -XX:OnError=/datadog-lib/continuousprofiler/tmp/dd_crash_uploader.sh -XX:ErrorFile=/datadog-lib/continuousprofiler/tmp/hs_err_pid_%p.log",
			wantErr:        false,
		},
		{
			name: "JAVA_TOOL_OPTIONS set via ValueFrom",
			pod:  fakePodWithEnvFieldRefValue("java-pod", "JAVA_TOOL_OPTIONS", "path"),
			libsToInject: []libInfo{
				{
					lang:  "java",
					image: "gcr.io/datadoghq/dd-lib-java-init:v1",
				},
			},
			wantErr: true,
		},
		{
			name: "nominal case: js",
			pod:  fakePod("js-pod"),
			libsToInject: []libInfo{
				{
					lang:  "js",
					image: "gcr.io/datadoghq/dd-lib-js-init:v1",
				},
			},
			expectedEnvKey: "NODE_OPTIONS",
			expectedEnvVal: " --require=/datadog-lib/node_modules/dd-trace/init",
			wantErr:        false,
		},
		{
			name: "NODE_OPTIONS not empty",
			pod:  fakePodWithEnvValue("js-pod", "NODE_OPTIONS", "predefined"),
			libsToInject: []libInfo{
				{
					lang:  "js",
					image: "gcr.io/datadoghq/dd-lib-js-init:v1",
				},
			},
			expectedEnvKey: "NODE_OPTIONS",
			expectedEnvVal: "predefined --require=/datadog-lib/node_modules/dd-trace/init",
			wantErr:        false,
		},
		{
			name: "NODE_OPTIONS set via ValueFrom",
			pod:  fakePodWithEnvFieldRefValue("js-pod", "NODE_OPTIONS", "path"),
			libsToInject: []libInfo{
				{
					lang:  "js",
					image: "gcr.io/datadoghq/dd-lib-js-init:v1",
				},
			},
			wantErr: true,
		},
		{
			name: "nominal case: python",
			pod:  fakePod("python-pod"),
			libsToInject: []libInfo{
				{
					lang:  "python",
					image: "gcr.io/datadoghq/dd-lib-python-init:v1",
				},
			},
			expectedEnvKey: "PYTHONPATH",
			expectedEnvVal: "/datadog-lib/",
			wantErr:        false,
		},
		{
			name: "PYTHONPATH not empty",
			pod:  fakePodWithEnvValue("python-pod", "PYTHONPATH", "predefined"),
			libsToInject: []libInfo{
				{
					lang:  "python",
					image: "gcr.io/datadoghq/dd-lib-python-init:v1",
				},
			},
			expectedEnvKey: "PYTHONPATH",
			expectedEnvVal: "/datadog-lib/:predefined",
			wantErr:        false,
		},
		{
			name: "PYTHONPATH set via ValueFrom",
			pod:  fakePodWithEnvFieldRefValue("python-pod", "PYTHONPATH", "path"),
			libsToInject: []libInfo{
				{
					lang:  "python",
					image: "gcr.io/datadoghq/dd-lib-python-init:v1",
				},
			},
			wantErr: true,
		},
		{
			name: "Unknown language",
			pod:  fakePod("unknown-pod"),
			libsToInject: []libInfo{
				{
					lang:  "unknown",
					image: "gcr.io/datadoghq/dd-lib-unknown-init:v1",
				},
			},
			wantErr: true,
		},
		{
			name: "nominal case: dotnet",
			pod:  fakePod("dotnet-pod"),
			libsToInject: []libInfo{
				{
					lang:  "dotnet",
					image: "gcr.io/datadoghq/dd-lib-dotnet-init:v1",
				},
			},
			expectedEnvKey: "CORECLR_PROFILER",
			expectedEnvVal: "{846F5F1C-F9AE-4B07-969E-05C26BC060D8}",
			wantErr:        false,
		},
		{
			name: "CORECLR_ENABLE_PROFILING not empty",
			pod:  fakePodWithEnvValue("dotnet-pod", "CORECLR_PROFILER", "predefined"),
			libsToInject: []libInfo{
				{
					lang:  "dotnet",
					image: "gcr.io/datadoghq/dd-lib-dotnet-init:v1",
				},
			},
			expectedEnvKey: "CORECLR_PROFILER",
			expectedEnvVal: "{846F5F1C-F9AE-4B07-969E-05C26BC060D8}",
			wantErr:        false,
		},
		{
			name: "CORECLR_ENABLE_PROFILING set via ValueFrom",
			pod:  fakePodWithEnvFieldRefValue("dotnet-pod", "CORECLR_PROFILER", "path"),
			libsToInject: []libInfo{
				{
					lang:  "dotnet",
					image: "gcr.io/datadoghq/dd-lib-dotnet-init:v1",
				},
			},
			wantErr: true,
		},
		{
			name: "nominal case: ruby",
			pod:  fakePod("ruby-pod"),
			libsToInject: []libInfo{
				{
					lang:  "ruby",
					image: "gcr.io/datadoghq/dd-lib-ruby-init:v1",
				},
			},
			expectedEnvKey: "RUBYOPT",
			expectedEnvVal: " -r/datadog-lib/auto_inject",
			wantErr:        false,
		},
		{
			name: "RUBYOPT not empty",
			pod:  fakePodWithEnvValue("ruby-pod", "RUBYOPT", "predefined"),
			libsToInject: []libInfo{
				{
					lang:  "ruby",
					image: "gcr.io/datadoghq/dd-lib-ruby-init:v1",
				},
			},
			expectedEnvKey: "RUBYOPT",
			expectedEnvVal: "predefined -r/datadog-lib/auto_inject",
			wantErr:        false,
		},
		{
			name: "RUBYOPT set via ValueFrom",
			pod:  fakePodWithEnvFieldRefValue("ruby-pod", "RUBYOPT", "path"),
			libsToInject: []libInfo{
				{
					lang:  "ruby",
					image: "gcr.io/datadoghq/dd-lib-ruby-init:v1",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := injectAutoInstruConfig(tt.pod, tt.libsToInject, false)
			require.False(t, (err != nil) != tt.wantErr)
			if err != nil {
				return
			}
			assertLibReq(t, tt.pod, tt.libsToInject[0].lang, tt.libsToInject[0].image, tt.expectedEnvKey, tt.expectedEnvVal)
		})
	}
}

func assertLibReq(t *testing.T, pod *corev1.Pod, lang language, image, envKey, envVal string) {
	// Empty dir volume
	volumeFound := false
	for _, volume := range pod.Spec.Volumes {
		if volume.Name == "datadog-auto-instrumentation" {
			require.NotNil(t, volume.VolumeSource.EmptyDir)
			volumeFound = true
			break
		}
	}
	require.True(t, volumeFound)

	// Init container
	initContainerFound := false
	for _, container := range pod.Spec.InitContainers {
		if container.Name == fmt.Sprintf("datadog-lib-%s-init", lang) {
			require.Equal(t, image, container.Image)
			require.Equal(t, []string{"sh", "copy-lib.sh", "/datadog-lib"}, container.Command)
			require.Equal(t, "datadog-auto-instrumentation", container.VolumeMounts[0].Name)
			require.Equal(t, "/datadog-lib", container.VolumeMounts[0].MountPath)
			initContainerFound = true
			break
		}
	}
	require.True(t, initContainerFound)

	// App container
	container := pod.Spec.Containers[0]
	require.Equal(t, "datadog-auto-instrumentation", container.VolumeMounts[0].Name)
	require.Equal(t, "/datadog-lib", container.VolumeMounts[0].MountPath)
	envFound := false
	for _, env := range container.Env {
		if env.Name == envKey {
			require.Equal(t, envVal, env.Value)
			envFound = true
			break
		}
	}
	require.True(t, envFound)
}

func TestExtractLibInfo(t *testing.T) {
	var mockConfig *config.MockConfig
	tests := []struct {
		name                 string
		pod                  *corev1.Pod
		containerRegistry    string
		expectedLibsToInject []libInfo
		setupConfig          func()
	}{
		{
			name:              "java",
			pod:               fakePodWithAnnotation("admission.datadoghq.com/java-lib.version", "v1"),
			containerRegistry: "registry",
			expectedLibsToInject: []libInfo{
				{
					lang:  "java",
					image: "registry/dd-lib-java-init:v1",
				},
			},
		},
		{
			name:              "js",
			pod:               fakePodWithAnnotation("admission.datadoghq.com/js-lib.version", "v1"),
			containerRegistry: "registry",
			expectedLibsToInject: []libInfo{
				{
					lang:  "js",
					image: "registry/dd-lib-js-init:v1",
				},
			},
		},
		{
			name:              "python",
			pod:               fakePodWithAnnotation("admission.datadoghq.com/python-lib.version", "v1"),
			containerRegistry: "registry",
			expectedLibsToInject: []libInfo{
				{
					lang:  "python",
					image: "registry/dd-lib-python-init:v1",
				},
			},
		},
		{
			name:              "custom",
			pod:               fakePodWithAnnotation("admission.datadoghq.com/java-lib.custom-image", "custom/image"),
			containerRegistry: "registry",
			expectedLibsToInject: []libInfo{
				{
					lang:  "java",
					image: "custom/image",
				},
			},
		},
		{
			name:                 "unknown",
			pod:                  fakePodWithAnnotation("admission.datadoghq.com/unknown-lib.version", "v1"),
			containerRegistry:    "registry",
			expectedLibsToInject: []libInfo{},
		},
		{
			name: "java and js",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"admission.datadoghq.com/java-lib.version": "v1",
						"admission.datadoghq.com/js-lib.version":   "v1",
					},
				},
			},
			containerRegistry: "registry",
			expectedLibsToInject: []libInfo{
				{
					lang:  "java",
					image: "registry/dd-lib-java-init:v1",
				},
				{
					lang:  "js",
					image: "registry/dd-lib-js-init:v1",
				},
			},
		},
		{
			name: "java and js on specific containers",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"admission.datadoghq.com/java-app.java-lib.version": "v1",
						"admission.datadoghq.com/node-app.js-lib.version":   "v1",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "java-app",
						},
						{
							Name: "node-app",
						},
					},
				},
			},
			containerRegistry: "registry",
			expectedLibsToInject: []libInfo{
				{
					ctrName: "java-app",
					lang:    "java",
					image:   "registry/dd-lib-java-init:v1",
				},
				{
					ctrName: "node-app",
					lang:    "js",
					image:   "registry/dd-lib-js-init:v1",
				},
			},
		},
		{
			name:              "ruby",
			pod:               fakePodWithAnnotation("admission.datadoghq.com/ruby-lib.version", "v1"),
			containerRegistry: "registry",
			expectedLibsToInject: []libInfo{
				{
					lang:  "ruby",
					image: "registry/dd-lib-ruby-init:v1",
				},
			},
		},
		{
			name:              "all",
			pod:               fakePodWithAnnotation("admission.datadoghq.com/all-lib.version", "latest"),
			containerRegistry: "registry",
			expectedLibsToInject: []libInfo{ // TODO: Add new entry when a new language is supported
				{
					lang:  "java",
					image: "registry/dd-lib-java-init:latest",
				},
				{
					lang:  "js",
					image: "registry/dd-lib-js-init:latest",
				},
				{
					lang:  "python",
					image: "registry/dd-lib-python-init:latest",
				},
				{
					lang:  "dotnet",
					image: "registry/dd-lib-dotnet-init:latest",
				},
				{
					lang:  "ruby",
					image: "registry/dd-lib-ruby-init:latest",
				},
			},
		},
		{
			name: "java + all",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"admission.datadoghq.com/all-lib.version":  "latest",
						"admission.datadoghq.com/java-lib.version": "v1",
					},
				},
			},
			containerRegistry: "registry",
			expectedLibsToInject: []libInfo{
				{
					lang:  "java",
					image: "registry/dd-lib-java-init:v1",
				},
			},
		},
		{
			name:              "all with unsupported version",
			pod:               fakePodWithAnnotation("admission.datadoghq.com/all-lib.version", "unsupported"),
			containerRegistry: "registry",
			expectedLibsToInject: []libInfo{
				{
					lang:  "java",
					image: "registry/dd-lib-java-init:latest",
				},
				{
					lang:  "js",
					image: "registry/dd-lib-js-init:latest",
				},
				{
					lang:  "python",
					image: "registry/dd-lib-python-init:latest",
				},
				{
					lang:  "dotnet",
					image: "registry/dd-lib-dotnet-init:latest",
				},
				{
					lang:  "ruby",
					image: "registry/dd-lib-ruby-init:latest",
				},
			},
			setupConfig: func() { mockConfig.SetWithoutSource("apm_config.instrumentation.enabled", false) },
		},
		{
			name:              "single step instrumentation with no pinned versions",
			pod:               fakePodWithNamespaceAndLabel("ns", "", ""),
			containerRegistry: "registry",
			expectedLibsToInject: []libInfo{
				{
					lang:  "java",
					image: "registry/dd-lib-java-init:latest",
				},
				{
					lang:  "js",
					image: "registry/dd-lib-js-init:latest",
				},
				{
					lang:  "python",
					image: "registry/dd-lib-python-init:latest",
				},
				{
					lang:  "dotnet",
					image: "registry/dd-lib-dotnet-init:latest",
				},
				{
					lang:  "ruby",
					image: "registry/dd-lib-ruby-init:latest",
				},
			},
			setupConfig: func() { mockConfig.SetWithoutSource("apm_config.instrumentation.enabled", true) },
		},
		{
			name:              "single step instrumentation with pinned java version",
			pod:               fakePodWithNamespaceAndLabel("ns", "", ""),
			containerRegistry: "registry",
			expectedLibsToInject: []libInfo{
				{
					lang:  "java",
					image: "registry/dd-lib-java-init:v1.20.0",
				},
				{
					lang:  "js",
					image: "registry/dd-lib-js-init:latest",
				},
				{
					lang:  "python",
					image: "registry/dd-lib-python-init:latest",
				},
				{
					lang:  "dotnet",
					image: "registry/dd-lib-dotnet-init:latest",
				},
				{
					lang:  "ruby",
					image: "registry/dd-lib-ruby-init:latest",
				},
			},
			setupConfig: func() {
				mockConfig.SetWithoutSource("apm_config.instrumentation.enabled", true)
				mockConfig.SetWithoutSource("apm_config.instrumentation.lib_versions", map[string]string{"java": "v1.20.0"})
			},
		},
		{
			name:              "single step instrumentation with pinned java and python versions",
			pod:               fakePodWithNamespaceAndLabel("ns", "", ""),
			containerRegistry: "registry",
			expectedLibsToInject: []libInfo{
				{
					lang:  "java",
					image: "registry/dd-lib-java-init:v1.20.0",
				},
				{
					lang:  "js",
					image: "registry/dd-lib-js-init:latest",
				},
				{
					lang:  "python",
					image: "registry/dd-lib-python-init:v1.19.0",
				},
				{
					lang:  "dotnet",
					image: "registry/dd-lib-dotnet-init:latest",
				},
				{
					lang:  "ruby",
					image: "registry/dd-lib-ruby-init:latest",
				},
			},
			setupConfig: func() {
				mockConfig.SetWithoutSource("apm_config.instrumentation.enabled", true)
				mockConfig.SetWithoutSource("apm_config.instrumentation.lib_versions", map[string]string{"java": "v1.20.0", "python": "v1.19.0"})
			},
		},
		{
			name:              "single step instrumentation with pinned java version and java annotation",
			pod:               fakePodWithAnnotation("admission.datadoghq.com/java-lib.version", "v1"),
			containerRegistry: "registry",
			expectedLibsToInject: []libInfo{
				{
					lang:  "java",
					image: "registry/dd-lib-java-init:v1",
				},
				{
					lang:  "js",
					image: "registry/dd-lib-js-init:latest",
				},
				{
					lang:  "python",
					image: "registry/dd-lib-python-init:latest",
				},
				{
					lang:  "dotnet",
					image: "registry/dd-lib-dotnet-init:latest",
				},
				{
					lang:  "ruby",
					image: "registry/dd-lib-ruby-init:latest",
				},
			},
			setupConfig: func() {
				mockConfig.SetWithoutSource("apm_config.instrumentation.enabled", true)
				mockConfig.SetWithoutSource("apm_config.instrumentation.lib_versions", map[string]string{"java": "v1.20.0"})
				mockConfig.SetWithoutSource("admission_controller.mutate_unlabelled", true)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConfig = config.Mock(t)
			mockConfig.SetWithoutSource("admission_controller.mutate_unlabelled", true)
			if tt.setupConfig != nil {
				tt.setupConfig()
			}
			libsToInject, _ := extractLibInfo(tt.pod, tt.containerRegistry)
			require.ElementsMatch(t, tt.expectedLibsToInject, libsToInject)
		})
	}
}

func TestInjectLibConfig(t *testing.T) {
	tests := []struct {
		name         string
		pod          *corev1.Pod
		lang         language
		wantErr      bool
		expectedEnvs []corev1.EnvVar
	}{
		{
			name:    "nominal case",
			pod:     fakePodWithAnnotation("admission.datadoghq.com/java-lib.config.v1", `{"version":1,"service_language":"java","runtime_metrics_enabled":true,"tracing_rate_limit":50}`),
			lang:    java,
			wantErr: false,
			expectedEnvs: []corev1.EnvVar{
				{
					Name:  "DD_RUNTIME_METRICS_ENABLED",
					Value: "true",
				},
				{
					Name:  "DD_TRACE_RATE_LIMIT",
					Value: "50",
				},
			},
		},
		{
			name:    "inject all case",
			pod:     fakePodWithAnnotation("admission.datadoghq.com/all-lib.config.v1", `{"version":1,"service_language":"all","runtime_metrics_enabled":true,"tracing_rate_limit":50}`),
			lang:    "all",
			wantErr: false,
			expectedEnvs: []corev1.EnvVar{
				{
					Name:  "DD_RUNTIME_METRICS_ENABLED",
					Value: "true",
				},
				{
					Name:  "DD_TRACE_RATE_LIMIT",
					Value: "50",
				},
			},
		},
		{
			name:         "invalid json",
			pod:          fakePodWithAnnotation("admission.datadoghq.com/java-lib.config.v1", "invalid"),
			lang:         java,
			wantErr:      true,
			expectedEnvs: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := injectLibConfig(tt.pod, tt.lang)
			require.False(t, (err != nil) != tt.wantErr)
			if err != nil {
				return
			}
			container := tt.pod.Spec.Containers[0]
			envCount := 0
			for _, expectEnv := range tt.expectedEnvs {
				for _, contEnv := range container.Env {
					if expectEnv.Name == contEnv.Name {
						require.Equal(t, expectEnv.Value, contEnv.Value)
						envCount++
						break
					}
				}
			}
			require.Equal(t, len(tt.expectedEnvs), envCount)
		})
	}
}

func TestInjectLibInitContainer(t *testing.T) {
	tests := []struct {
		name    string
		cpu     string
		mem     string
		pod     *corev1.Pod
		image   string
		lang    language
		wantErr bool
	}{
		{
			name:    "no resources",
			pod:     fakePod("java-pod"),
			image:   "gcr.io/datadoghq/dd-lib-java-init:v1",
			lang:    java,
			wantErr: false,
		},
		{
			name:    "with resources",
			pod:     fakePod("java-pod"),
			cpu:     "100m",
			mem:     "500",
			image:   "gcr.io/datadoghq/dd-lib-java-init:v1",
			lang:    java,
			wantErr: false,
		},
		{
			name:    "cpu only",
			pod:     fakePod("java-pod"),
			cpu:     "200m",
			image:   "gcr.io/datadoghq/dd-lib-java-init:v1",
			lang:    java,
			wantErr: false,
		},
		{
			name:    "memory only",
			pod:     fakePod("java-pod"),
			mem:     "512Mi",
			image:   "gcr.io/datadoghq/dd-lib-java-init:v1",
			lang:    java,
			wantErr: false,
		},
		{
			name:    "with invalid resources",
			pod:     fakePod("java-pod"),
			cpu:     "foo",
			image:   "gcr.io/datadoghq/dd-lib-java-init:v1",
			lang:    java,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := config.Mock(t)
			if tt.cpu != "" {
				conf.SetWithoutSource("admission_controller.auto_instrumentation.init_resources.cpu", tt.cpu)
			}
			if tt.mem != "" {
				conf.SetWithoutSource("admission_controller.auto_instrumentation.init_resources.memory", tt.mem)
			}
			err := injectLibInitContainer(tt.pod, tt.image, tt.lang)
			if (err != nil) != tt.wantErr {
				t.Errorf("injectLibInitContainer() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			require.Len(t, tt.pod.Spec.InitContainers, 1)
			if tt.cpu != "" {
				req := tt.pod.Spec.InitContainers[0].Resources.Requests[corev1.ResourceCPU]
				lim := tt.pod.Spec.InitContainers[0].Resources.Limits[corev1.ResourceCPU]
				require.Equal(t, tt.cpu, req.String())
				require.Equal(t, tt.cpu, lim.String())
			}
			if tt.mem != "" {
				req := tt.pod.Spec.InitContainers[0].Resources.Requests[corev1.ResourceMemory]
				lim := tt.pod.Spec.InitContainers[0].Resources.Limits[corev1.ResourceMemory]
				require.Equal(t, tt.mem, req.String())
				require.Equal(t, tt.mem, lim.String())
			}
		})
	}
}

func expBasicConfig() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "DD_TRACE_RATE_LIMIT",
			Value: "100",
		},
		{
			Name:  "DD_TRACE_SAMPLE_RATE",
			Value: "1.00",
		},
		{
			Name:  "DD_RUNTIME_METRICS_ENABLED",
			Value: "true",
		},
		{
			Name:  "DD_TRACE_HEALTH_METRICS_ENABLED",
			Value: "true",
		},
		{
			Name:  "DD_TRACE_ENABLED",
			Value: "true",
		},
		{
			Name:  "DD_LOGS_INJECTION",
			Value: "true",
		},
	}
}

func TestInjectAll(t *testing.T) {
	wantAll := map[string]libInfo{
		"java":   {lang: java, image: "gcr.io/datadoghq/dd-lib-java-init:latest"},
		"js":     {lang: js, image: "gcr.io/datadoghq/dd-lib-js-init:latest"},
		"python": {lang: python, image: "gcr.io/datadoghq/dd-lib-python-init:latest"},
		"dotnet": {lang: dotnet, image: "gcr.io/datadoghq/dd-lib-dotnet-init:latest"},
		"ruby":   {lang: ruby, image: "gcr.io/datadoghq/dd-lib-ruby-init:latest"},
	}
	tests := []struct {
		name             string
		ns               string
		targetNamespaces []string
		want             map[string]libInfo
	}{
		{
			name:             "nominal",
			ns:               "targeted",
			targetNamespaces: []string{"targeted"},
			want:             wantAll,
		},
		{
			name:             "many",
			ns:               "targeted",
			targetNamespaces: []string{"targeted", "foo", "bar"},
			want:             wantAll,
		},
		{
			name:             "no match",
			ns:               "not-targeted",
			targetNamespaces: []string{"foo", "bar"},
			want:             map[string]libInfo{},
		},
		{
			name:             "empty target",
			ns:               "targeted",
			targetNamespaces: []string{},
			want:             map[string]libInfo{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targetNamespaces = tt.targetNamespaces
			got := getAllLibsToInject(tt.ns, "gcr.io/datadoghq")
			require.EqualValues(t, tt.want, got)
		})
	}
}

func injectAllEnvs() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "PYTHONPATH",
			Value: "/datadog-lib/",
		},
		{
			Name:  "RUBYOPT",
			Value: " -r/datadog-lib/auto_inject",
		},
		{
			Name:  "NODE_OPTIONS",
			Value: " --require=/datadog-lib/node_modules/dd-trace/init",
		},
		{
			Name:  "JAVA_TOOL_OPTIONS",
			Value: " -javaagent:/datadog-lib/dd-java-agent.jar -XX:OnError=/datadog-lib/continuousprofiler/tmp/dd_crash_uploader.sh -XX:ErrorFile=/datadog-lib/continuousprofiler/tmp/hs_err_pid_%p.log",
		},
		{
			Name:  "DD_DOTNET_TRACER_HOME",
			Value: "/datadog-lib",
		},
		{
			Name:  "CORECLR_ENABLE_PROFILING",
			Value: "1",
		},
		{
			Name:  "CORECLR_PROFILER",
			Value: "{846F5F1C-F9AE-4B07-969E-05C26BC060D8}",
		},
		{
			Name:  "CORECLR_PROFILER_PATH",
			Value: "/datadog-lib/Datadog.Trace.ClrProfiler.Native.so",
		},
		{
			Name:  "DD_TRACE_LOG_DIRECTORY",
			Value: "/datadog-lib/logs",
		},
		{
			Name:  "LD_PRELOAD",
			Value: "/datadog-lib/continuousprofiler/Datadog.Linux.ApiWrapper.x64.so",
		},
	}
}

func TestInjectAutoInstrumentation(t *testing.T) {
	var mockConfig *config.MockConfig
	uuid := uuid.New().String()
	installTime := strconv.FormatInt(time.Now().Unix(), 10)
	t.Setenv("DD_INSTRUMENTATION_INSTALL_ID", uuid)
	t.Setenv("DD_INSTRUMENTATION_INSTALL_TIME", installTime)
	tests := []struct {
		name         string
		pod          *corev1.Pod
		expectedEnvs []corev1.EnvVar
		wantErr      bool
		setupConfig  func()
	}{
		{
			name: "inject all",
			pod: fakePodWithParent(
				"ns",
				map[string]string{
					"admission.datadoghq.com/all-lib.version":   "latest",
					"admission.datadoghq.com/all-lib.config.v1": `{"version":1,"runtime_metrics_enabled":true,"tracing_rate_limit":50,"tracing_sampling_rate":0.3}`,
				},
				map[string]string{
					"admission.datadoghq.com/enabled": "true",
				},
				[]corev1.EnvVar{},
				"replicaset",
				"deployment-1234",
			),
			expectedEnvs: []corev1.EnvVar{
				{
					Name:  "DD_INSTRUMENTATION_INSTALL_TYPE",
					Value: "k8s_lib_injection",
				},
				{
					Name:  "DD_INSTRUMENTATION_INSTALL_TIME",
					Value: installTime,
				},
				{
					Name:  "DD_INSTRUMENTATION_INSTALL_ID",
					Value: uuid,
				},
				{
					Name:  "DD_RUNTIME_METRICS_ENABLED",
					Value: "true",
				},
				{
					Name:  "DD_TRACE_RATE_LIMIT",
					Value: "50",
				},
				{
					Name:  "DD_TRACE_SAMPLE_RATE",
					Value: "0.30",
				},
				{
					Name:  "PYTHONPATH",
					Value: "/datadog-lib/",
				},
				{
					Name:  "RUBYOPT",
					Value: " -r/datadog-lib/auto_inject",
				},
				{
					Name:  "NODE_OPTIONS",
					Value: " --require=/datadog-lib/node_modules/dd-trace/init",
				},
				{
					Name:  "JAVA_TOOL_OPTIONS",
					Value: " -javaagent:/datadog-lib/dd-java-agent.jar -XX:OnError=/datadog-lib/continuousprofiler/tmp/dd_crash_uploader.sh -XX:ErrorFile=/datadog-lib/continuousprofiler/tmp/hs_err_pid_%p.log",
				},
				{
					Name:  "DD_DOTNET_TRACER_HOME",
					Value: "/datadog-lib",
				},
				{
					Name:  "CORECLR_ENABLE_PROFILING",
					Value: "1",
				},
				{
					Name:  "CORECLR_PROFILER",
					Value: "{846F5F1C-F9AE-4B07-969E-05C26BC060D8}",
				},
				{
					Name:  "CORECLR_PROFILER_PATH",
					Value: "/datadog-lib/Datadog.Trace.ClrProfiler.Native.so",
				},
				{
					Name:  "DD_TRACE_LOG_DIRECTORY",
					Value: "/datadog-lib/logs",
				},
				{
					Name:  "LD_PRELOAD",
					Value: "/datadog-lib/continuousprofiler/Datadog.Linux.ApiWrapper.x64.so",
				},
			},
			wantErr:     false,
			setupConfig: func() {},
		},
		{
			name: "inject java",
			pod: fakePodWithParent(
				"ns",
				map[string]string{
					"admission.datadoghq.com/java-lib.version":   "latest",
					"admission.datadoghq.com/java-lib.config.v1": `{"version":1,"tracing_sampling_rate":0.3}`,
				},
				map[string]string{
					"admission.datadoghq.com/enabled": "true",
				},
				[]corev1.EnvVar{},
				"replicaset",
				"deployment-1234",
			),
			expectedEnvs: []corev1.EnvVar{
				{
					Name:  "DD_INSTRUMENTATION_INSTALL_TYPE",
					Value: "k8s_lib_injection",
				},
				{
					Name:  "DD_INSTRUMENTATION_INSTALL_TIME",
					Value: installTime,
				},
				{
					Name:  "DD_INSTRUMENTATION_INSTALL_ID",
					Value: uuid,
				},
				{
					Name:  "DD_TRACE_SAMPLE_RATE",
					Value: "0.30",
				},
				{
					Name:  "JAVA_TOOL_OPTIONS",
					Value: " -javaagent:/datadog-lib/dd-java-agent.jar -XX:OnError=/datadog-lib/continuousprofiler/tmp/dd_crash_uploader.sh -XX:ErrorFile=/datadog-lib/continuousprofiler/tmp/hs_err_pid_%p.log",
				},
			},
			wantErr: false,
		},
		{
			name: "inject python",
			pod:  fakePodWithParent("ns", map[string]string{"admission.datadoghq.com/python-lib.version": "latest", "admission.datadoghq.com/python-lib.config.v1": `{"version":1,"tracing_sampling_rate":0.3}`}, map[string]string{"admission.datadoghq.com/enabled": "true"}, []corev1.EnvVar{}, "", ""),
			expectedEnvs: []corev1.EnvVar{
				{
					Name:  "DD_TRACE_SAMPLE_RATE",
					Value: "0.30",
				},
				{
					Name:  "PYTHONPATH",
					Value: "/datadog-lib/",
				},
				{
					Name:  "DD_INSTRUMENTATION_INSTALL_TYPE",
					Value: "k8s_lib_injection",
				},
				{
					Name:  "DD_INSTRUMENTATION_INSTALL_TIME",
					Value: installTime,
				},
				{
					Name:  "DD_INSTRUMENTATION_INSTALL_ID",
					Value: uuid,
				},
			},
			wantErr: false,
			setupConfig: func() {
			},
		},
		{
			name: "inject node",
			pod:  fakePodWithParent("ns", map[string]string{"admission.datadoghq.com/js-lib.version": "latest", "admission.datadoghq.com/js-lib.config.v1": `{"version":1,"tracing_sampling_rate":0.3}`}, map[string]string{"admission.datadoghq.com/enabled": "true"}, []corev1.EnvVar{}, "", ""),
			expectedEnvs: []corev1.EnvVar{
				{
					Name:  "DD_TRACE_SAMPLE_RATE",
					Value: "0.30",
				},
				{
					Name:  "NODE_OPTIONS",
					Value: " --require=/datadog-lib/node_modules/dd-trace/init",
				},
				{
					Name:  "DD_INSTRUMENTATION_INSTALL_TYPE",
					Value: "k8s_lib_injection",
				},
				{
					Name:  "DD_INSTRUMENTATION_INSTALL_TIME",
					Value: installTime,
				},
				{
					Name:  "DD_INSTRUMENTATION_INSTALL_ID",
					Value: uuid,
				},
			},
			wantErr: false,
			setupConfig: func() {
			},
		},
		{
			name: "inject library and all",
			pod: fakePodWithParent(
				"ns",
				map[string]string{
					"admission.datadoghq.com/all-lib.version":   "latest",
					"admission.datadoghq.com/all-lib.config.v1": `{"version":1,"runtime_metrics_enabled":true,"tracing_rate_limit":50,"tracing_sampling_rate":0.3}`,
					"admission.datadoghq.com/js-lib.version":    "v1.10",
					"admission.datadoghq.com/js-lib.config.v1":  `{"version":1,"tracing_sampling_rate":0.4}`,
				},
				map[string]string{
					"admission.datadoghq.com/enabled": "true",
				},
				[]corev1.EnvVar{},
				"",
				"",
			),
			expectedEnvs: []corev1.EnvVar{
				{
					Name:  "DD_RUNTIME_METRICS_ENABLED",
					Value: "true",
				},
				{
					Name:  "DD_TRACE_RATE_LIMIT",
					Value: "50",
				},
				{
					Name:  "DD_TRACE_SAMPLE_RATE",
					Value: "0.40",
				},
				{
					Name:  "NODE_OPTIONS",
					Value: " --require=/datadog-lib/node_modules/dd-trace/init",
				},
				{
					Name:  "DD_INSTRUMENTATION_INSTALL_TYPE",
					Value: "k8s_lib_injection",
				},
				{
					Name:  "DD_INSTRUMENTATION_INSTALL_TIME",
					Value: installTime,
				},
				{
					Name:  "DD_INSTRUMENTATION_INSTALL_ID",
					Value: uuid,
				},
			},
			wantErr: false,
		},
		{
			name: "inject library and all no library version",
			pod: fakePodWithParent(
				"ns",
				map[string]string{
					"admission.datadoghq.com/all-lib.version":   "latest",
					"admission.datadoghq.com/all-lib.config.v1": `{"version":1,"runtime_metrics_enabled":true,"tracing_rate_limit":50,"tracing_sampling_rate":0.3}`,
					"admission.datadoghq.com/js-lib.config.v1":  `{"version":1,"tracing_sampling_rate":0.4}`,
				},
				map[string]string{
					"admission.datadoghq.com/enabled": "true",
				},
				[]corev1.EnvVar{},
				"",
				"",
			),
			expectedEnvs: []corev1.EnvVar{
				{
					Name:  "DD_INSTRUMENTATION_INSTALL_TYPE",
					Value: "k8s_lib_injection",
				},
				{
					Name:  "DD_INSTRUMENTATION_INSTALL_TIME",
					Value: installTime,
				},
				{
					Name:  "DD_INSTRUMENTATION_INSTALL_ID",
					Value: uuid,
				},
				{
					Name:  "DD_RUNTIME_METRICS_ENABLED",
					Value: "true",
				},
				{
					Name:  "DD_TRACE_RATE_LIMIT",
					Value: "50",
				},
				{
					Name:  "PYTHONPATH",
					Value: "/datadog-lib/",
				},
				{
					Name:  "RUBYOPT",
					Value: " -r/datadog-lib/auto_inject",
				},
				{
					Name:  "NODE_OPTIONS",
					Value: " --require=/datadog-lib/node_modules/dd-trace/init",
				},
				{
					Name:  "JAVA_TOOL_OPTIONS",
					Value: " -javaagent:/datadog-lib/dd-java-agent.jar -XX:OnError=/datadog-lib/continuousprofiler/tmp/dd_crash_uploader.sh -XX:ErrorFile=/datadog-lib/continuousprofiler/tmp/hs_err_pid_%p.log",
				},
				{
					Name:  "DD_DOTNET_TRACER_HOME",
					Value: "/datadog-lib",
				},
				{
					Name:  "CORECLR_ENABLE_PROFILING",
					Value: "1",
				},
				{
					Name:  "CORECLR_PROFILER",
					Value: "{846F5F1C-F9AE-4B07-969E-05C26BC060D8}",
				},
				{
					Name:  "CORECLR_PROFILER_PATH",
					Value: "/datadog-lib/Datadog.Trace.ClrProfiler.Native.so",
				},
				{
					Name:  "DD_TRACE_LOG_DIRECTORY",
					Value: "/datadog-lib/logs",
				},
				{
					Name:  "LD_PRELOAD",
					Value: "/datadog-lib/continuousprofiler/Datadog.Linux.ApiWrapper.x64.so",
				},
				{
					Name:  "DD_TRACE_SAMPLE_RATE",
					Value: "0.40",
				},
			},
			wantErr: false,
		},
		{
			name: "inject all error - bad json",
			pod: fakePodWithParent(
				"ns",
				map[string]string{
					// TODO: we might not want to be injecting the libraries if the config is malformed
					"admission.datadoghq.com/all-lib.version":   "latest",
					"admission.datadoghq.com/all-lib.config.v1": `{"version":1,"runtime_metrics_enabled":true,`,
				},
				map[string]string{
					"admission.datadoghq.com/enabled": "true",
				},
				[]corev1.EnvVar{},
				"",
				"",
			),
			expectedEnvs: []corev1.EnvVar{
				{
					Name:  "DD_INSTRUMENTATION_INSTALL_TYPE",
					Value: "k8s_lib_injection",
				},
				{
					Name:  "DD_INSTRUMENTATION_INSTALL_TIME",
					Value: installTime,
				},
				{
					Name:  "DD_INSTRUMENTATION_INSTALL_ID",
					Value: uuid,
				},
				{
					Name:  "PYTHONPATH",
					Value: "/datadog-lib/",
				},
				{
					Name:  "RUBYOPT",
					Value: " -r/datadog-lib/auto_inject",
				},
				{
					Name:  "NODE_OPTIONS",
					Value: " --require=/datadog-lib/node_modules/dd-trace/init",
				},
				{
					Name:  "JAVA_TOOL_OPTIONS",
					Value: " -javaagent:/datadog-lib/dd-java-agent.jar -XX:OnError=/datadog-lib/continuousprofiler/tmp/dd_crash_uploader.sh -XX:ErrorFile=/datadog-lib/continuousprofiler/tmp/hs_err_pid_%p.log",
				},
				{
					Name:  "DD_DOTNET_TRACER_HOME",
					Value: "/datadog-lib",
				},
				{
					Name:  "CORECLR_ENABLE_PROFILING",
					Value: "1",
				},
				{
					Name:  "CORECLR_PROFILER",
					Value: "{846F5F1C-F9AE-4B07-969E-05C26BC060D8}",
				},
				{
					Name:  "CORECLR_PROFILER_PATH",
					Value: "/datadog-lib/Datadog.Trace.ClrProfiler.Native.so",
				},
				{
					Name:  "DD_TRACE_LOG_DIRECTORY",
					Value: "/datadog-lib/logs",
				},
				{
					Name:  "LD_PRELOAD",
					Value: "/datadog-lib/continuousprofiler/Datadog.Linux.ApiWrapper.x64.so",
				},
			},
			wantErr: true,
		},
		{
			name: "inject java bad json",
			pod: fakePodWithParent(
				"ns",
				map[string]string{
					"admission.datadoghq.com/java-lib.version":   "latest",
					"admission.datadoghq.com/java-lib.config.v1": `{"version":1,"runtime_metrics_enabled":true,`,
				},
				map[string]string{
					"admission.datadoghq.com/enabled": "true",
				},
				[]corev1.EnvVar{},
				"",
				"",
			),
			expectedEnvs: []corev1.EnvVar{
				{
					Name:  "DD_INSTRUMENTATION_INSTALL_TYPE",
					Value: "k8s_lib_injection",
				},
				{
					Name:  "DD_INSTRUMENTATION_INSTALL_TIME",
					Value: installTime,
				},
				{
					Name:  "DD_INSTRUMENTATION_INSTALL_ID",
					Value: uuid,
				},
				{
					Name:  "JAVA_TOOL_OPTIONS",
					Value: " -javaagent:/datadog-lib/dd-java-agent.jar -XX:OnError=/datadog-lib/continuousprofiler/tmp/dd_crash_uploader.sh -XX:ErrorFile=/datadog-lib/continuousprofiler/tmp/hs_err_pid_%p.log",
				},
			},
			wantErr: true,
		},
		{
			name: "inject with enabled false",
			pod: fakePodWithParent(
				"ns",
				map[string]string{
					"admission.datadoghq.com/java-lib.version":   "latest",
					"admission.datadoghq.com/java-lib.config.v1": `{"version":1,"runtime_metrics_enabled":true,`,
				},
				map[string]string{
					"admission.datadoghq.com/enabled": "false",
				},
				[]corev1.EnvVar{},
				"",
				"",
			),
			expectedEnvs: []corev1.EnvVar{
				{
					Name:  "DD_INSTRUMENTATION_INSTALL_TIME",
					Value: installTime,
				},
				{
					Name:  "DD_INSTRUMENTATION_INSTALL_ID",
					Value: uuid,
				},
			},
			wantErr: false,
		},
		{
			name: "Single Step Instrumentation: user configuration is respected",
			pod: fakePodWithParent("ns", map[string]string{}, map[string]string{}, []corev1.EnvVar{
				{
					Name:  "DD_SERVICE",
					Value: "user-deployment",
				},
				{
					Name:  "DD_TRACE_ENABLED",
					Value: "false",
				},
				{
					Name:  "DD_RUNTIME_METRICS_ENABLED",
					Value: "false",
				},
				{
					Name:  "DD_TRACE_HEALTH_METRICS_ENABLED",
					Value: "false",
				},
				{
					Name:  "DD_TRACE_SAMPLE_RATE",
					Value: "0.5",
				},
				{
					Name:  "DD_TRACE_RATE_LIMIT",
					Value: "2",
				},
				{
					Name:  "DD_LOGS_INJECTION",
					Value: "false",
				},
			}, "replicaset", "test-deployment-123"),
			expectedEnvs: append(injectAllEnvs(), []corev1.EnvVar{
				{
					Name:  "DD_INSTRUMENTATION_INSTALL_TYPE",
					Value: "k8s_single_step",
				},
				{
					Name:  "DD_INSTRUMENTATION_INSTALL_TIME",
					Value: installTime,
				},
				{
					Name:  "DD_INSTRUMENTATION_INSTALL_ID",
					Value: uuid,
				},
				{
					Name:  "DD_SERVICE",
					Value: "user-deployment",
				},
				{
					Name:  "DD_TRACE_ENABLED",
					Value: "false",
				},
				{
					Name:  "DD_RUNTIME_METRICS_ENABLED",
					Value: "false",
				},
				{
					Name:  "DD_TRACE_HEALTH_METRICS_ENABLED",
					Value: "false",
				},
				{
					Name:  "DD_TRACE_SAMPLE_RATE",
					Value: "0.5",
				},
				{
					Name:  "DD_TRACE_RATE_LIMIT",
					Value: "2",
				},
				{
					Name:  "DD_LOGS_INJECTION",
					Value: "false",
				},
			}...),
			wantErr:     false,
			setupConfig: func() { mockConfig.SetWithoutSource("apm_config.instrumentation.enabled", true) },
		},
		{
			name: "Single Step Instrumentation: disable with label",
			pod: fakePodWithParent(
				"ns",
				map[string]string{}, map[string]string{
					"admission.datadoghq.com/enabled": "false",
				}, []corev1.EnvVar{}, "replicaset", "test-deployment-123"),
			expectedEnvs: []corev1.EnvVar{
				{
					Name:  "DD_INSTRUMENTATION_INSTALL_TIME",
					Value: installTime,
				},
				{
					Name:  "DD_INSTRUMENTATION_INSTALL_ID",
					Value: uuid,
				},
			},
			wantErr:     false,
			setupConfig: func() { mockConfig.SetWithoutSource("apm_config.instrumentation.enabled", true) },
		},
		{
			name: "Single Step Instrumentation: default service name for ReplicaSet",
			pod: fakePodWithParent(
				"ns",
				map[string]string{},
				map[string]string{},
				[]corev1.EnvVar{},
				"replicaset",
				"test-deployment-123",
			),
			expectedEnvs: append(append(injectAllEnvs(), expBasicConfig()...), corev1.EnvVar{
				Name:  "DD_SERVICE",
				Value: "test-deployment",
			},
				corev1.EnvVar{
					Name:  "DD_INSTRUMENTATION_INSTALL_TYPE",
					Value: "k8s_single_step",
				},
				corev1.EnvVar{
					Name:  "DD_INSTRUMENTATION_INSTALL_TIME",
					Value: installTime,
				},
				corev1.EnvVar{
					Name:  "DD_INSTRUMENTATION_INSTALL_ID",
					Value: uuid,
				},
			),
			wantErr:     false,
			setupConfig: func() { mockConfig.SetWithoutSource("apm_config.instrumentation.enabled", true) },
		},
		{
			name: "Single Step Instrumentation: default service name for StatefulSet",
			pod: fakePodWithParent(
				"ns",
				map[string]string{},
				map[string]string{},
				[]corev1.EnvVar{},
				"statefulset",
				"test-statefulset-123",
			),
			expectedEnvs: append(append(injectAllEnvs(), expBasicConfig()...), corev1.EnvVar{
				Name:  "DD_SERVICE",
				Value: "test-statefulset-123",
			},
				corev1.EnvVar{
					Name:  "DD_INSTRUMENTATION_INSTALL_TYPE",
					Value: "k8s_single_step",
				},
				corev1.EnvVar{
					Name:  "DD_INSTRUMENTATION_INSTALL_TIME",
					Value: installTime,
				},
				corev1.EnvVar{
					Name:  "DD_INSTRUMENTATION_INSTALL_ID",
					Value: uuid,
				},
			),
			wantErr:     false,
			setupConfig: func() { mockConfig.SetWithoutSource("apm_config.instrumentation.enabled", true) },
		},
		{
			name: "Single Step Instrumentation: default service name (disabled)",
			pod:  fakePodWithParent("ns", map[string]string{}, map[string]string{}, []corev1.EnvVar{}, "replicaset", "test-deployment-123"),
			expectedEnvs: []corev1.EnvVar{
				{
					Name:  "DD_INSTRUMENTATION_INSTALL_TIME",
					Value: installTime,
				},
				{
					Name:  "DD_INSTRUMENTATION_INSTALL_ID",
					Value: uuid,
				},
			},
			wantErr:     false,
			setupConfig: func() { mockConfig.SetWithoutSource("apm_config.instrumentation.enabled", false) },
		},
		{
			name: "Single Step Instrumentation: disabled namespaces should not be instrumented",
			pod:  fakePodWithParent("ns", map[string]string{}, map[string]string{}, []corev1.EnvVar{}, "replicaset", "test-app-123"),
			expectedEnvs: []corev1.EnvVar{
				{
					Name:  "DD_INSTRUMENTATION_INSTALL_TIME",
					Value: installTime,
				},
				{
					Name:  "DD_INSTRUMENTATION_INSTALL_ID",
					Value: uuid,
				},
			},
			wantErr: false,
			setupConfig: func() {
				mockConfig.SetWithoutSource("apm_config.instrumentation.enabled", true)
				mockConfig.SetWithoutSource("apm_config.instrumentation.disabled_namespaces", []string{"ns"})
			},
		},
		{
			name: "Single Step Instrumentation: enabled namespaces should be instrumented",
			pod: fakePodWithParent(
				"ns",
				map[string]string{},
				map[string]string{},
				[]corev1.EnvVar{{Name: "DD_INSTRUMENTATION_INSTALL_TYPE", Value: "k8s_single_step"}},
				"replicaset", "test-app-123",
			),
			expectedEnvs: append(append(injectAllEnvs(), expBasicConfig()...), corev1.EnvVar{
				Name:  "DD_SERVICE",
				Value: "test-app",
			},
				corev1.EnvVar{
					Name:  "DD_INSTRUMENTATION_INSTALL_TYPE",
					Value: "k8s_single_step",
				},
				corev1.EnvVar{
					Name:  "DD_INSTRUMENTATION_INSTALL_TIME",
					Value: installTime,
				},
				corev1.EnvVar{
					Name:  "DD_INSTRUMENTATION_INSTALL_ID",
					Value: uuid,
				},
			),
			wantErr: false,
			setupConfig: func() {
				mockConfig.SetWithoutSource("apm_config.instrumentation.enabled", false)
				mockConfig.SetWithoutSource("apm_config.instrumentation.enabled_namespaces", []string{"ns"})
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockConfig = config.Mock(t)
			if tt.setupConfig != nil {
				tt.setupConfig()
			}
			err := injectAutoInstrumentation(tt.pod, "", fake.NewSimpleDynamicClient(scheme))
			require.False(t, (err != nil) != tt.wantErr)

			container := tt.pod.Spec.Containers[0]
			for _, contEnv := range container.Env {
				found := false
				for _, expectEnv := range tt.expectedEnvs {
					if expectEnv.Name == contEnv.Name {
						found = true
						break
					}
				}
				if !found {
					require.Failf(t, "Unexpected env var injected in container", contEnv.Name)
				}
			}
			for _, expectEnv := range tt.expectedEnvs {
				found := false
				for _, contEnv := range container.Env {
					if expectEnv.Name == contEnv.Name {
						found = true
						break
					}
				}
				if !found {
					require.Failf(t, "Unexpected env var injected in container", expectEnv.Name)
				}
			}

			envCount := 0
			for _, contEnv := range container.Env {
				for _, expectEnv := range tt.expectedEnvs {
					if expectEnv.Name == contEnv.Name {
						require.Equal(t, expectEnv.Value, contEnv.Value)
						envCount++
						break
					}
				}
			}
			require.Equal(t, len(tt.expectedEnvs), envCount)
		})
		t.Cleanup(func() {
			os.Unsetenv("DD_INSTRUMENTATION_INSTALL_ID")
			os.Unsetenv("DD_INSTRUMENTATION_INSTALL_TIME")
		})
	}
}
