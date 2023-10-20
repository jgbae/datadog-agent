// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package profile

import (
	"bufio"
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cihub/seelog"
	assert "github.com/stretchr/testify/require"

	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/util/log"

	"github.com/DataDog/datadog-agent/pkg/networkdevice/profile/profiledefinition"
)

func getMetricFromProfile(p profiledefinition.ProfileDefinition, metricName string) *profiledefinition.MetricsConfig {
	for _, m := range p.Metrics {
		if m.Symbol.Name == metricName {
			return &m
		}
	}
	return nil
}

func Test_getDefaultProfilesDefinitionFiles(t *testing.T) {
	SetConfdPathAndCleanProfiles()
	actualProfileConfig, err := getDefaultProfilesDefinitionFiles()
	assert.Nil(t, err)

	confdPath := config.Datadog.GetString("confd_path")
	expectedProfileConfig := ProfileConfigMap{
		"f5-big-ip": {
			DefinitionFile: filepath.Join(confdPath, "snmp.d", "profiles", "f5-big-ip.yaml"),
			IsUserProfile:  true,
		},
		"another_profile": {
			DefinitionFile: filepath.Join(confdPath, "snmp.d", "profiles", "another_profile.yaml"),
			IsUserProfile:  true,
		},
	}

	assert.Equal(t, expectedProfileConfig, actualProfileConfig)
}

func Test_loadProfiles(t *testing.T) {
	defaultTestConfdPath, _ := filepath.Abs(filepath.Join("..", "test", "conf.d"))
	config.Datadog.Set("confd_path", defaultTestConfdPath)
	defaultProfilesDef, err := getDefaultProfilesDefinitionFiles()
	assert.Nil(t, err)

	profilesWithInvalidExtendConfdPath, _ := filepath.Abs(filepath.Join("..", "test", "invalid_ext.d"))
	invalidCyclicConfdPath, _ := filepath.Abs(filepath.Join("..", "test", "invalid_cyclic.d"))

	profileWithInvalidExtends, _ := filepath.Abs(filepath.Join("..", "test", "test_profiles", "profile_with_invalid_extends.yaml"))
	invalidYamlProfile, _ := filepath.Abs(filepath.Join("..", "test", "test_profiles", "invalid_yaml_file.yaml"))
	validationErrorProfile, _ := filepath.Abs(filepath.Join("..", "test", "test_profiles", "validation_error.yaml"))
	type logCount struct {
		log   string
		count int
	}
	tests := []struct {
		name                  string
		confdPath             string
		inputProfileConfigMap ProfileConfigMap
		expectedProfileDefMap ProfileConfigMap
		expectedIncludeErrors []string
		expectedLogs          []logCount
	}{
		{
			name:                  "ok case",
			confdPath:             defaultTestConfdPath,
			inputProfileConfigMap: defaultProfilesDef,
			expectedProfileDefMap: FixtureProfileDefinitionMap(),
			expectedIncludeErrors: []string{},
		},
		{
			name: "failed to read profile",
			inputProfileConfigMap: ProfileConfigMap{
				"f5-big-ip": {
					DefinitionFile: filepath.Join(string(filepath.Separator), "does", "not", "exist"),
					IsUserProfile:  true,
				},
			},
			expectedProfileDefMap: ProfileConfigMap{},
			expectedLogs: []logCount{
				{"[WARN] loadProfiles: failed to read profile definition `f5-big-ip`: failed to read file", 1},
			},
		},
		{
			name: "invalid extends",
			inputProfileConfigMap: ProfileConfigMap{
				"f5-big-ip": {
					DefinitionFile: profileWithInvalidExtends,
					IsUserProfile:  true,
				},
			},
			expectedProfileDefMap: ProfileConfigMap{},
			expectedLogs: []logCount{
				{"[WARN] loadProfiles: failed to expand profile `f5-big-ip`: failed to read file", 1},
			},
		},
		{
			name:      "invalid recursive extends",
			confdPath: profilesWithInvalidExtendConfdPath,
			inputProfileConfigMap: ProfileConfigMap{
				"f5-big-ip": {
					DefinitionFile: "f5-big-ip.yaml",
				},
			},
			expectedProfileDefMap: ProfileConfigMap{},
			expectedLogs: []logCount{
				{"[WARN] loadProfiles: failed to expand profile `f5-big-ip`", 1},
				{"invalid.yaml", 2},
			},
		},
		{
			name:      "invalid cyclic extends",
			confdPath: invalidCyclicConfdPath,
			inputProfileConfigMap: ProfileConfigMap{
				"f5-big-ip": {
					DefinitionFile: "f5-big-ip.yaml",
				},
			},
			expectedProfileDefMap: ProfileConfigMap{},
			expectedLogs: []logCount{
				{"[WARN] loadProfiles: failed to expand profile `f5-big-ip`: cyclic profile extend detected, `_extend1.yaml` has already been extended, extendsHistory=`[_extend1.yaml _extend2.yaml]", 1},
			},
		},
		{
			name: "invalid yaml profile",
			inputProfileConfigMap: ProfileConfigMap{
				"f5-big-ip": {
					DefinitionFile: invalidYamlProfile,
				},
			},
			expectedProfileDefMap: ProfileConfigMap{},
			expectedLogs: []logCount{
				{"failed to read profile definition `f5-big-ip`: failed to unmarshall", 1},
			},
		},
		{
			name: "validation error profile",
			inputProfileConfigMap: ProfileConfigMap{
				"f5-big-ip": {
					DefinitionFile: validationErrorProfile,
				},
			},
			expectedProfileDefMap: ProfileConfigMap{},
			expectedLogs: []logCount{
				{"cannot compile `match` (`global_metric_tags[\\w)(\\w+)`)", 1},
				{"cannot compile `match` (`table_match[\\w)`)", 1},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b bytes.Buffer
			w := bufio.NewWriter(&b)
			l, err := seelog.LoggerFromWriterWithMinLevelAndFormat(w, seelog.DebugLvl, "[%LEVEL] %FuncShort: %Msg")
			assert.Nil(t, err)
			log.SetupLogger(l, "debug")

			config.Datadog.Set("confd_path", tt.confdPath)

			profiles, err := loadProfiles(tt.inputProfileConfigMap)
			for _, errorMsg := range tt.expectedIncludeErrors {
				assert.Contains(t, err.Error(), errorMsg)
			}

			w.Flush()
			logs := b.String()

			for _, aLogCount := range tt.expectedLogs {
				assert.Equal(t, aLogCount.count, strings.Count(logs, aLogCount.log), logs)
			}

			for i, profile := range profiles {
				profiledefinition.NormalizeMetrics(profile.Definition.Metrics)
				profile.DefinitionFile = ""
				profiles[i] = profile
			}

			assert.Equal(t, tt.expectedProfileDefMap, profiles)
		})
	}
}

func Test_getMostSpecificOid(t *testing.T) {
	tests := []struct {
		name           string
		oids           []string
		expectedOid    string
		expectedErrror error
	}{
		{
			"one",
			[]string{"1.2.3.4"},
			"1.2.3.4",
			nil,
		},
		{
			"error on empty oids",
			[]string{},
			"",
			fmt.Errorf("cannot get most specific oid from empty list of oids"),
		},
		{
			"error on parsing",
			[]string{"a.1.2.3"},
			"",
			fmt.Errorf("error parsing part `a` for pattern `a.1.2.3`: strconv.Atoi: parsing \"a\": invalid syntax"),
		},
		{
			"most lengthy",
			[]string{"1.3.4", "1.3.4.1.2"},
			"1.3.4.1.2",
			nil,
		},
		{
			"wild card 1",
			[]string{"1.3.4.*", "1.3.4.1"},
			"1.3.4.1",
			nil,
		},
		{
			"wild card 2",
			[]string{"1.3.4.1", "1.3.4.*"},
			"1.3.4.1",
			nil,
		},
		{
			"sample oids",
			[]string{"1.3.6.1.4.1.3375.2.1.3.4.43", "1.3.6.1.4.1.8072.3.2.10"},
			"1.3.6.1.4.1.3375.2.1.3.4.43",
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oid, err := getMostSpecificOid(tt.oids)
			assert.Equal(t, tt.expectedErrror, err)
			assert.Equal(t, tt.expectedOid, oid)
		})
	}
}

func Test_resolveProfileDefinitionPath(t *testing.T) {
	defaultTestConfdPath, _ := filepath.Abs(filepath.Join("..", "test", "user_profiles.d"))
	config.Datadog.Set("confd_path", defaultTestConfdPath)

	absPath, _ := filepath.Abs(filepath.Join("tmp", "myfile.yaml"))
	tests := []struct {
		name               string
		definitionFilePath string
		expectedPath       string
	}{
		{
			name:               "abs path",
			definitionFilePath: absPath,
			expectedPath:       absPath,
		},
		{
			name:               "relative path with default profile",
			definitionFilePath: "p2.yaml",
			expectedPath:       filepath.Join(config.Datadog.Get("confd_path").(string), "snmp.d", "default_profiles", "p2.yaml"),
		},
		{
			name:               "relative path with user profile",
			definitionFilePath: "p3.yaml",
			expectedPath:       filepath.Join(config.Datadog.Get("confd_path").(string), "snmp.d", "profiles", "p3.yaml"),
		},
		{
			name:               "relative path with user profile precedence",
			definitionFilePath: "p1.yaml",
			expectedPath:       filepath.Join(config.Datadog.Get("confd_path").(string), "snmp.d", "profiles", "p1.yaml"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := resolveProfileDefinitionPath(tt.definitionFilePath)
			assert.Equal(t, tt.expectedPath, path)
		})
	}
}

func Test_loadDefaultProfiles(t *testing.T) {
	SetConfdPathAndCleanProfiles()
	SetGlobalProfileConfigMap(nil)
	defaultProfiles, err := loadYamlProfiles()
	assert.Nil(t, err)
	defaultProfiles2, err := loadYamlProfiles()
	assert.Nil(t, err)

	assert.Equal(t, fmt.Sprintf("%p", defaultProfiles), fmt.Sprintf("%p", defaultProfiles2))
}

func Test_loadDefaultProfiles_withUserProfiles(t *testing.T) {
	defaultTestConfdPath, _ := filepath.Abs(filepath.Join("..", "test", "user_profiles.d"))
	SetGlobalProfileConfigMap(nil)
	config.Datadog.Set("confd_path", defaultTestConfdPath)

	defaultProfiles, err := loadYamlProfiles()
	assert.Nil(t, err)

	assert.Len(t, defaultProfiles, 4)
	assert.NotNil(t, defaultProfiles)

	p1 := defaultProfiles["p1"].Definition // user p1 overrides datadog p1
	p2 := defaultProfiles["p2"].Definition // datadog p2
	p3 := defaultProfiles["p3"].Definition // user p3
	p4 := defaultProfiles["p4"].Definition // user p3

	assert.Equal(t, "p1_user", p1.Device.Vendor) // overrides datadog p1 profile
	assert.NotNil(t, getMetricFromProfile(p1, "p1_metric_override"))

	assert.Equal(t, "p2_datadog", p2.Device.Vendor)
	assert.NotNil(t, getMetricFromProfile(p2, "p2_metric"))

	assert.Equal(t, "p3_user", p3.Device.Vendor)
	assert.NotNil(t, getMetricFromProfile(p3, "p3_metric"))

	assert.Equal(t, "p4_user", p4.Device.Vendor)
	assert.NotNil(t, getMetricFromProfile(p4, "p4_user_metric"))
	assert.NotNil(t, getMetricFromProfile(p4, "p4_default_metric"))
}

func Test_loadDefaultProfiles_invalidDir(t *testing.T) {
	invalidPath, _ := filepath.Abs(filepath.Join(".", "tmp", "invalidPath"))
	config.Datadog.Set("confd_path", invalidPath)
	SetGlobalProfileConfigMap(nil)

	defaultProfiles, err := loadYamlProfiles()
	assert.Nil(t, err)
	assert.Len(t, defaultProfiles, 0)
}

func Test_loadDefaultProfiles_invalidExtendProfile(t *testing.T) {
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	l, err := seelog.LoggerFromWriterWithMinLevelAndFormat(w, seelog.DebugLvl, "[%LEVEL] %FuncShort: %Msg")
	assert.Nil(t, err)
	log.SetupLogger(l, "debug")

	profilesWithInvalidExtendConfdPath, _ := filepath.Abs(filepath.Join("..", "test", "invalid_ext.d"))
	config.Datadog.Set("confd_path", profilesWithInvalidExtendConfdPath)
	SetGlobalProfileConfigMap(nil)

	defaultProfiles, err := loadYamlProfiles()

	w.Flush()
	logs := b.String()
	assert.Nil(t, err)

	assert.Equal(t, 1, strings.Count(logs, "[WARN] loadProfilesV2: failed to expand profile `f5-big-ip"), logs)
	assert.Equal(t, ProfileConfigMap{}, defaultProfiles)
}

func Test_loadDefaultProfiles_validAndInvalidProfiles(t *testing.T) {
	// Valid profiles should be returned even if some profiles are invalid
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	l, err := seelog.LoggerFromWriterWithMinLevelAndFormat(w, seelog.DebugLvl, "[%LEVEL] %FuncShort: %Msg")
	assert.Nil(t, err)
	log.SetupLogger(l, "debug")

	profilesWithInvalidExtendConfdPath, _ := filepath.Abs(filepath.Join("..", "test", "valid_invalid.d"))
	config.Datadog.Set("confd_path", profilesWithInvalidExtendConfdPath)
	SetGlobalProfileConfigMap(nil)

	defaultProfiles, err := loadYamlProfiles()

	for _, profile := range defaultProfiles {
		profiledefinition.NormalizeMetrics(profile.Definition.Metrics)
	}

	w.Flush()
	logs := b.String()
	assert.Nil(t, err)

	assert.Equal(t, 1, strings.Count(logs, "[WARN] loadProfilesV2: failed to read profile definition `f5-invalid`"), logs)
	assert.Contains(t, defaultProfiles, "f5-big-ip")
	assert.NotContains(t, defaultProfiles, "f5-invalid")
}

func Test_mergeProfileDefinition(t *testing.T) {
	okBaseDefinition := profiledefinition.ProfileDefinition{
		Metrics: []profiledefinition.MetricsConfig{
			{Symbol: profiledefinition.SymbolConfig{OID: "1.1", Name: "metric1"}, MetricType: profiledefinition.ProfileMetricTypeGauge},
		},
		MetricTags: []profiledefinition.MetricTagConfig{
			{
				Tag:    "tag1",
				Symbol: profiledefinition.SymbolConfigCompat{OID: "2.1", Name: "tagName1"},
			},
		},
		Metadata: profiledefinition.MetadataConfig{
			"device": {
				Fields: map[string]profiledefinition.MetadataField{
					"vendor": {
						Value: "f5",
					},
					"description": {
						Symbol: profiledefinition.SymbolConfig{
							OID:  "1.3.6.1.2.1.1.1.0",
							Name: "sysDescr",
						},
					},
				},
			},
			"interface": {
				Fields: map[string]profiledefinition.MetadataField{
					"admin_status": {
						Symbol: profiledefinition.SymbolConfig{

							OID:  "1.3.6.1.2.1.2.2.1.7",
							Name: "ifAdminStatus",
						},
					},
				},
				IDTags: profiledefinition.MetricTagConfigList{
					{
						Tag: "alias",
						Symbol: profiledefinition.SymbolConfigCompat{
							OID:  "1.3.6.1.2.1.31.1.1.1.1",
							Name: "ifAlias",
						},
					},
				},
			},
		},
	}
	emptyBaseDefinition := profiledefinition.ProfileDefinition{}
	okTargetDefinition := profiledefinition.ProfileDefinition{
		Metrics: []profiledefinition.MetricsConfig{
			{Symbol: profiledefinition.SymbolConfig{OID: "1.2", Name: "metric2"}, MetricType: profiledefinition.ProfileMetricTypeGauge},
		},
		MetricTags: []profiledefinition.MetricTagConfig{
			{
				Tag:    "tag2",
				Symbol: profiledefinition.SymbolConfigCompat{OID: "2.2", Name: "tagName2"},
			},
		},
		Metadata: profiledefinition.MetadataConfig{
			"device": {
				Fields: map[string]profiledefinition.MetadataField{
					"name": {
						Symbol: profiledefinition.SymbolConfig{
							OID:  "1.3.6.1.2.1.1.5.0",
							Name: "sysName",
						},
					},
				},
			},
			"interface": {
				Fields: map[string]profiledefinition.MetadataField{
					"oper_status": {
						Symbol: profiledefinition.SymbolConfig{
							OID:  "1.3.6.1.2.1.2.2.1.8",
							Name: "ifOperStatus",
						},
					},
				},
				IDTags: profiledefinition.MetricTagConfigList{
					{
						Tag: "interface",
						Symbol: profiledefinition.SymbolConfigCompat{
							OID:  "1.3.6.1.2.1.31.1.1.1.1",
							Name: "ifName",
						},
					},
				},
			},
		},
	}
	tests := []struct {
		name               string
		targetDefinition   profiledefinition.ProfileDefinition
		baseDefinition     profiledefinition.ProfileDefinition
		expectedDefinition profiledefinition.ProfileDefinition
	}{
		{
			name:             "merge case",
			baseDefinition:   CopyProfileDefinition(okBaseDefinition),
			targetDefinition: CopyProfileDefinition(okTargetDefinition),
			expectedDefinition: profiledefinition.ProfileDefinition{
				Metrics: []profiledefinition.MetricsConfig{
					{Symbol: profiledefinition.SymbolConfig{OID: "1.2", Name: "metric2"}, MetricType: profiledefinition.ProfileMetricTypeGauge},
					{Symbol: profiledefinition.SymbolConfig{OID: "1.1", Name: "metric1"}, MetricType: profiledefinition.ProfileMetricTypeGauge},
				},
				MetricTags: []profiledefinition.MetricTagConfig{
					{
						Tag:    "tag2",
						Symbol: profiledefinition.SymbolConfigCompat{OID: "2.2", Name: "tagName2"},
					},
					{
						Tag:    "tag1",
						Symbol: profiledefinition.SymbolConfigCompat{OID: "2.1", Name: "tagName1"},
					},
				},
				Metadata: profiledefinition.MetadataConfig{
					"device": {
						Fields: map[string]profiledefinition.MetadataField{
							"vendor": {
								Value: "f5",
							},
							"name": {
								Symbol: profiledefinition.SymbolConfig{
									OID:  "1.3.6.1.2.1.1.5.0",
									Name: "sysName",
								},
							},
							"description": {
								Symbol: profiledefinition.SymbolConfig{
									OID:  "1.3.6.1.2.1.1.1.0",
									Name: "sysDescr",
								},
							},
						},
					},
					"interface": {
						Fields: map[string]profiledefinition.MetadataField{
							"oper_status": {
								Symbol: profiledefinition.SymbolConfig{
									OID:  "1.3.6.1.2.1.2.2.1.8",
									Name: "ifOperStatus",
								},
							},
							"admin_status": {
								Symbol: profiledefinition.SymbolConfig{

									OID:  "1.3.6.1.2.1.2.2.1.7",
									Name: "ifAdminStatus",
								},
							},
						},
						IDTags: profiledefinition.MetricTagConfigList{
							{
								Tag: "interface",
								Symbol: profiledefinition.SymbolConfigCompat{
									OID:  "1.3.6.1.2.1.31.1.1.1.1",
									Name: "ifName",
								},
							},
							{
								Tag: "alias",
								Symbol: profiledefinition.SymbolConfigCompat{
									OID:  "1.3.6.1.2.1.31.1.1.1.1",
									Name: "ifAlias",
								},
							},
						},
					},
				},
			},
		},
		{
			name:             "empty base definition",
			baseDefinition:   CopyProfileDefinition(emptyBaseDefinition),
			targetDefinition: CopyProfileDefinition(okTargetDefinition),
			expectedDefinition: profiledefinition.ProfileDefinition{
				Metrics: []profiledefinition.MetricsConfig{
					{Symbol: profiledefinition.SymbolConfig{OID: "1.2", Name: "metric2"}, MetricType: profiledefinition.ProfileMetricTypeGauge},
				},
				MetricTags: []profiledefinition.MetricTagConfig{
					{
						Tag:    "tag2",
						Symbol: profiledefinition.SymbolConfigCompat{OID: "2.2", Name: "tagName2"},
					},
				},
				Metadata: profiledefinition.MetadataConfig{
					"device": {
						Fields: map[string]profiledefinition.MetadataField{
							"name": {
								Symbol: profiledefinition.SymbolConfig{
									OID:  "1.3.6.1.2.1.1.5.0",
									Name: "sysName",
								},
							},
						},
					},
					"interface": {
						Fields: map[string]profiledefinition.MetadataField{
							"oper_status": {
								Symbol: profiledefinition.SymbolConfig{
									OID:  "1.3.6.1.2.1.2.2.1.8",
									Name: "ifOperStatus",
								},
							},
						},
						IDTags: profiledefinition.MetricTagConfigList{
							{
								Tag: "interface",
								Symbol: profiledefinition.SymbolConfigCompat{
									OID:  "1.3.6.1.2.1.31.1.1.1.1",
									Name: "ifName",
								},
							},
						},
					},
				},
			},
		},
		{
			name:             "empty taget definition",
			baseDefinition:   CopyProfileDefinition(okBaseDefinition),
			targetDefinition: CopyProfileDefinition(emptyBaseDefinition),
			expectedDefinition: profiledefinition.ProfileDefinition{
				Metrics: []profiledefinition.MetricsConfig{
					{Symbol: profiledefinition.SymbolConfig{OID: "1.1", Name: "metric1"}, MetricType: profiledefinition.ProfileMetricTypeGauge},
				},
				MetricTags: []profiledefinition.MetricTagConfig{
					{
						Tag:    "tag1",
						Symbol: profiledefinition.SymbolConfigCompat{OID: "2.1", Name: "tagName1"},
					},
				},
				Metadata: profiledefinition.MetadataConfig{
					"device": {
						Fields: map[string]profiledefinition.MetadataField{
							"vendor": {
								Value: "f5",
							},
							"description": {
								Symbol: profiledefinition.SymbolConfig{
									OID:  "1.3.6.1.2.1.1.1.0",
									Name: "sysDescr",
								},
							},
						},
					},
					"interface": {
						Fields: map[string]profiledefinition.MetadataField{
							"admin_status": {
								Symbol: profiledefinition.SymbolConfig{

									OID:  "1.3.6.1.2.1.2.2.1.7",
									Name: "ifAdminStatus",
								},
							},
						},
						IDTags: profiledefinition.MetricTagConfigList{
							{
								Tag: "alias",
								Symbol: profiledefinition.SymbolConfigCompat{
									OID:  "1.3.6.1.2.1.31.1.1.1.1",
									Name: "ifAlias",
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mergeProfileDefinition(&tt.targetDefinition, &tt.baseDefinition)
			assert.Equal(t, tt.expectedDefinition.Metrics, tt.targetDefinition.Metrics)
			assert.Equal(t, tt.expectedDefinition.MetricTags, tt.targetDefinition.MetricTags)
			assert.Equal(t, tt.expectedDefinition.Metadata, tt.targetDefinition.Metadata)
		})
	}
}
