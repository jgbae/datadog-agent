// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package report

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/DataDog/datadog-agent/pkg/collector/corechecks/snmp/internal/common"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"github.com/cihub/seelog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/DataDog/datadog-agent/pkg/aggregator/mocksender"

	"github.com/DataDog/datadog-agent/pkg/networkdevice/profile/profiledefinition"
	"github.com/DataDog/datadog-agent/pkg/snmp/snmpintegration"

	"github.com/DataDog/datadog-agent/pkg/collector/corechecks/snmp/internal/valuestore"
)

const (
	mockDeviceID          string = "namespace:deviceIP"
	fullIndex             string = "9"
	mockInterfaceIDPrefix string = mockDeviceID + ":" + fullIndex
	ifSpeed               uint64 = 80 * (1e6)
)

// Mock interface rate map with previous metric samples for the interface with ifSpeed of 30
func interfaceRateMapWithPrevious() *InterfaceBandwidthState {
	return MockInterfaceRateMap(mockInterfaceIDPrefix, ifSpeed, ifSpeed, 30, 5, mockTimeNowNano15SecLater)
}

// Mock interface rate map with previous metric samples where the ifSpeed is taken from configuration files
func interfaceRateMapWithConfig() *InterfaceBandwidthState {
	return MockInterfaceRateMap(mockInterfaceIDPrefix, 160_000_000, 40_000_000, 20, 10, mockTimeNowNano15SecLater)
}

func Test_metricSender_calculateRate(t *testing.T) {
	tests := []struct {
		name             string
		symbols          []profiledefinition.SymbolConfig
		fullIndex        string
		values           *valuestore.ResultValueStore
		tags             []string
		interfaceConfigs []snmpintegration.InterfaceConfig
		expectedRate     float64
		usageValue       float64
	}{
		{
			name:      "snmp.ifBandwidthOutUsage.Rate ifHCInOctets Gauge submitted",
			symbols:   []profiledefinition.SymbolConfig{{OID: "1.3.6.1.2.1.31.1.1.1.6", Name: "ifHCInOctets"}},
			fullIndex: "9",
			tags:      []string{"abc"},
			values: &valuestore.ResultValueStore{
				ColumnValues: valuestore.ColumnResultValuesType{
					// ifHCInOctets
					"1.3.6.1.2.1.31.1.1.1.6": map[string]valuestore.ResultValue{
						"9": {
							Value: 5000000.0,
						},
					},
					// ifHCOutOctets
					"1.3.6.1.2.1.31.1.1.1.10": map[string]valuestore.ResultValue{
						"9": {
							Value: 1000000.0,
						},
					},
					// ifHighSpeed
					"1.3.6.1.2.1.31.1.1.1.15": map[string]valuestore.ResultValue{
						"9": {
							Value: 80.0,
						},
					},
				},
			},
			// current usage value @ ts 30 for ifBandwidthInUsage
			// ((5000000 * 8) / (80 * 1000000)) * 100 = 50.0
			// previous usage value @ ts 15
			// ((3000000 * 8) / (80 * 1000000)) * 100 = 30.0
			// rate generated between ts 15 and 30
			// (50 - 30) / (30 - 15)
			expectedRate: 20.0 / 15.0,
			usageValue:   50,
		},
		{
			name:      "snmp.ifBandwidthOutUsage.Rate ifHCOutOctets submitted",
			symbols:   []profiledefinition.SymbolConfig{{OID: "1.3.6.1.2.1.31.1.1.1.10", Name: "ifHCOutOctets"}},
			fullIndex: "9",
			values: &valuestore.ResultValueStore{
				ColumnValues: valuestore.ColumnResultValuesType{
					// ifHCInOctets
					"1.3.6.1.2.1.31.1.1.1.6": map[string]valuestore.ResultValue{
						"9": {
							Value: 5000000.0,
						},
					},
					// ifHCOutOctets
					"1.3.6.1.2.1.31.1.1.1.10": map[string]valuestore.ResultValue{
						"9": {
							Value: 1000000.0,
						},
					},
					// ifHighSpeed
					"1.3.6.1.2.1.31.1.1.1.15": map[string]valuestore.ResultValue{
						"9": {
							Value: 80.0,
						},
					},
				},
			},
			// current usage value @ ts 30 for ifBandwidthOutUsage
			// ((1000000 * 8) / (80 * 1000000)) * 100 = 10.0
			// previous usage value @ ts 15
			// ((500000 * 8) / (80 * 1000000)) * 100 = 5.0
			// rate generated between ts 15 and 30
			// (10 - 5) / (30 - 15)
			expectedRate: 5.0 / 15.0,
			usageValue:   10,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := mocksender.NewMockSender("testID") // required to initiate aggregator
			sender.On("Gauge", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

			TimeNow = common.MockTimeNow
			ms := &MetricSender{
				sender:                  sender,
				deviceID:                mockDeviceID,
				interfaceConfigs:        tt.interfaceConfigs,
				interfaceBandwidthState: interfaceRateMapWithPrevious(),
			}

			usageName := bandwidthMetricNameToUsage[tt.symbols[0].Name]
			interfaceID := mockInterfaceIDPrefix + "." + usageName
			rate, err := ms.interfaceBandwidthState.calculateBandwidthUsageRate(interfaceID, ifSpeed, tt.usageValue)

			// Expect no errors
			assert.Nil(t, err)

			assert.Equal(t, tt.expectedRate, rate)

			// Check that the map was updated with current values for next check run
			assert.Equal(t, ifSpeed, ms.interfaceBandwidthState.state[interfaceID].ifSpeed)
			assert.Equal(t, tt.usageValue, ms.interfaceBandwidthState.state[interfaceID].previousSample)
			assert.Equal(t, mockTimeNowNano, ms.interfaceBandwidthState.state[interfaceID].previousTsNano)
		})
	}
}

func Test_metricSender_calculateRate_logs(t *testing.T) {
	tests := []struct {
		name               string
		symbols            []profiledefinition.SymbolConfig
		fullIndex          string
		values             *valuestore.ResultValueStore
		tags               []string
		interfaceConfigs   []snmpintegration.InterfaceConfig
		expectedLogMessage string
		expectedError      error
		usageValue         float64
		newIfSpeed         uint64
	}{
		{
			name:      "snmp.ifBandwidthInUsage.Rate ifHCInOctets erred",
			symbols:   []profiledefinition.SymbolConfig{{OID: "1.3.6.1.2.1.31.1.1.1.6", Name: "ifHCInOctets"}},
			fullIndex: "9",
			tags:      []string{"abc"},
			values: &valuestore.ResultValueStore{
				ColumnValues: valuestore.ColumnResultValuesType{
					// ifHCInOctets
					"1.3.6.1.2.1.31.1.1.1.6": map[string]valuestore.ResultValue{
						"9": {
							Value: 5000000.0,
						},
					},
					// ifHCOutOctets
					"1.3.6.1.2.1.31.1.1.1.10": map[string]valuestore.ResultValue{
						"9": {
							Value: 1000000.0,
						},
					},
					// ifHighSpeed
					"1.3.6.1.2.1.31.1.1.1.15": map[string]valuestore.ResultValue{
						"9": {
							Value: 100.0,
						},
					},
				},
			},
			expectedLogMessage: fmt.Sprintf("ifSpeed changed from %d to %d for device and interface %s, no rate emitted", ifSpeed, uint64(100)*(1e6), "namespace:deviceIP:9.ifBandwidthInUsage"),
			// ((5000000 * 8) / (100 * 1000000)) * 100 = 40.0
			usageValue: 40,
			newIfSpeed: uint64(100) * (1e6),
		},
		{
			name:      "snmp.ifBandwidthOutUsage.Rate ifHCOutOctets erred",
			symbols:   []profiledefinition.SymbolConfig{{OID: "1.3.6.1.2.1.31.1.1.1.10", Name: "ifHCOutOctets"}},
			fullIndex: "9",
			values: &valuestore.ResultValueStore{
				ColumnValues: valuestore.ColumnResultValuesType{
					// ifHCInOctets
					"1.3.6.1.2.1.31.1.1.1.6": map[string]valuestore.ResultValue{
						"9": {
							Value: 5000000.0,
						},
					},
					// ifHCOutOctets
					"1.3.6.1.2.1.31.1.1.1.10": map[string]valuestore.ResultValue{
						"9": {
							Value: 1000000.0,
						},
					},
					// ifHighSpeed
					"1.3.6.1.2.1.31.1.1.1.15": map[string]valuestore.ResultValue{
						"9": {
							Value: 100.0,
						},
					},
				},
			},
			expectedLogMessage: fmt.Sprintf("ifSpeed changed from %d to %d for device and interface %s, no rate emitted", ifSpeed, uint64(100)*(1e6), "namespace:deviceIP:9.ifBandwidthOutUsage"),
			// ((1000000 * 8) / (100 * 1000000)) * 100 = 8.0
			usageValue: 8,
			newIfSpeed: uint64(100) * (1e6),
		},
		{
			name:      "snmp.ifBandwidthInUsage.Rate ifHCInOctets error on negative rate",
			symbols:   []profiledefinition.SymbolConfig{{OID: "1.3.6.1.2.1.31.1.1.1.6", Name: "ifHCInOctets"}},
			fullIndex: "9",
			tags:      []string{"abc"},
			values: &valuestore.ResultValueStore{
				ColumnValues: valuestore.ColumnResultValuesType{
					// ifHCInOctets
					"1.3.6.1.2.1.31.1.1.1.6": map[string]valuestore.ResultValue{
						"9": {
							Value: 500000.0,
						},
					},
					// ifHCOutOctets
					"1.3.6.1.2.1.31.1.1.1.10": map[string]valuestore.ResultValue{
						"9": {
							Value: 1000000.0,
						},
					},
					// ifHighSpeed
					"1.3.6.1.2.1.31.1.1.1.15": map[string]valuestore.ResultValue{
						"9": {
							Value: 100.0,
						},
					},
				},
			},
			expectedError: fmt.Errorf("Rate value for device/interface %s is negative, discarding it", "namespace:deviceIP:9.ifBandwidthInUsage"),
			// ((500000 * 8) / (100 * 1000000)) * 100 = 4.0
			usageValue: 4,
			// keep it the same interface speed, testing if the rate is negative only
			newIfSpeed: uint64(80) * (1e6),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b bytes.Buffer
			w := bufio.NewWriter(&b)

			l, err := seelog.LoggerFromWriterWithMinLevelAndFormat(w, seelog.InfoLvl, "[%LEVEL] %FuncShort: %Msg")
			assert.Nil(t, err)
			log.SetupLogger(l, "InfoLvl")

			sender := mocksender.NewMockSender("testID") // required to initiate aggregator
			sender.On("Gauge", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

			TimeNow = common.MockTimeNow
			ms := &MetricSender{
				sender:                  sender,
				deviceID:                mockDeviceID,
				interfaceConfigs:        tt.interfaceConfigs,
				interfaceBandwidthState: interfaceRateMapWithPrevious(),
			}
			// conflicting ifSpeed from mocked saved state (80) in interfaceRateMap
			//newIfSpeed := uint64(100) * (1e6)

			for _, symbol := range tt.symbols {
				usageName := bandwidthMetricNameToUsage[symbol.Name]
				interfaceID := mockDeviceID + ":" + fullIndex + "." + usageName
				rate, err := ms.interfaceBandwidthState.calculateBandwidthUsageRate(interfaceID, tt.newIfSpeed, tt.usageValue)

				if tt.expectedError == nil {
					assert.Nil(t, err)
				} else {
					assert.Equal(t, tt.expectedError, err)
				}

				w.Flush()
				logs := b.String()

				assert.Contains(t, logs, tt.expectedLogMessage)
				assert.Equal(t, float64(0), rate)

				// Check that the map was updated with current values for next check run
				assert.Equal(t, tt.newIfSpeed, ms.interfaceBandwidthState.state[interfaceID].ifSpeed)
				assert.Equal(t, tt.usageValue, ms.interfaceBandwidthState.state[interfaceID].previousSample)
				assert.Equal(t, mockTimeNowNano, ms.interfaceBandwidthState.state[interfaceID].previousTsNano)
			}
		})
	}
}

func Test_metricSender_sendBandwidthUsageMetric(t *testing.T) {
	type Metric struct {
		name  string
		value float64
	}
	tests := []struct {
		name             string
		symbols          []profiledefinition.SymbolConfig
		fullIndex        string
		values           *valuestore.ResultValueStore
		tags             []string
		interfaceConfigs []snmpintegration.InterfaceConfig
		expectedMetric   []Metric
		expectedError    error
		rateMap          *InterfaceBandwidthState
	}{
		{
			name:      "snmp.ifBandwidthInUsage.Rate submitted",
			symbols:   []profiledefinition.SymbolConfig{{OID: "1.3.6.1.2.1.31.1.1.1.6", Name: "ifHCInOctets"}},
			fullIndex: "9",
			tags:      []string{"abc"},
			values: &valuestore.ResultValueStore{
				ColumnValues: valuestore.ColumnResultValuesType{
					// ifHCInOctets
					"1.3.6.1.2.1.31.1.1.1.6": map[string]valuestore.ResultValue{
						"9": {
							Value: 5000000.0,
						},
					},
					// ifHCOutOctets
					"1.3.6.1.2.1.31.1.1.1.10": map[string]valuestore.ResultValue{
						"9": {
							Value: 1000000.0,
						},
					},
					// ifHighSpeed
					"1.3.6.1.2.1.31.1.1.1.15": map[string]valuestore.ResultValue{
						"9": {
							Value: 80.0,
						},
					},
				},
			},
			expectedMetric: []Metric{
				// current usage value @ ts 30
				// ((5000000 * 8) / (80 * 1000000)) * 100 = 50.0
				// previous usage value @ ts 15
				// ((3000000 * 8) / (80 * 1000000)) * 100 = 30.0
				// rate generated between ts 15 and 30
				// (50 - 30) / (30 - 15)
				{"snmp.ifBandwidthInUsage.rate", 20.0 / 15.0},
			},
			rateMap: interfaceRateMapWithPrevious(),
		},
		{
			name:      "snmp.ifBandwidthOutUsage.Rate submitted",
			symbols:   []profiledefinition.SymbolConfig{{OID: "1.3.6.1.2.1.31.1.1.1.10", Name: "ifHCOutOctets"}},
			fullIndex: "9",
			values: &valuestore.ResultValueStore{
				ColumnValues: valuestore.ColumnResultValuesType{
					// ifHCInOctets
					"1.3.6.1.2.1.31.1.1.1.6": map[string]valuestore.ResultValue{
						"9": {
							Value: 5000000.0,
						},
					},
					// ifHCOutOctets
					"1.3.6.1.2.1.31.1.1.1.10": map[string]valuestore.ResultValue{
						"9": {
							Value: 1000000.0,
						},
					},
					// ifHighSpeed
					"1.3.6.1.2.1.31.1.1.1.15": map[string]valuestore.ResultValue{
						"9": {
							Value: 80.0,
						},
					},
				},
			},
			expectedMetric: []Metric{
				// current usage value @ ts 30
				// ((1000000 * 8) / (80 * 1000000)) * 100 = 10.0
				// previous usage value @ ts 15
				// ((500000 * 8) / (80 * 1000000)) * 100 = 5.0
				// rate generated between ts 15 and 30
				// (10 - 5) / (30 - 15)
				{"snmp.ifBandwidthOutUsage.rate", 5.0 / 15.0},
			},
			rateMap: interfaceRateMapWithPrevious(),
		},
		{
			name:      "not a bandwidth metric",
			symbols:   []profiledefinition.SymbolConfig{{OID: "1.3.6.1.2.1.31.1.1.1.99", Name: "notABandwidthMetric"}},
			fullIndex: "9",
			values: &valuestore.ResultValueStore{
				ColumnValues: valuestore.ColumnResultValuesType{},
			},
			expectedMetric: []Metric{},
			rateMap:        interfaceRateMapWithPrevious(),
		},
		{
			name:      "missing ifHighSpeed",
			symbols:   []profiledefinition.SymbolConfig{{OID: "1.3.6.1.2.1.31.1.1.1.6", Name: "ifHCInOctets"}},
			fullIndex: "9",
			values: &valuestore.ResultValueStore{
				ColumnValues: valuestore.ColumnResultValuesType{
					// ifHCInOctets
					"1.3.6.1.2.1.31.1.1.1.6": map[string]valuestore.ResultValue{
						"9": {
							Value: 5000000.0,
						},
					},
					// ifHCOutOctets
					"1.3.6.1.2.1.31.1.1.1.10": map[string]valuestore.ResultValue{
						"9": {
							Value: 1000000.0,
						},
					},
				},
			},
			expectedMetric: []Metric{},
			expectedError:  fmt.Errorf("bandwidth usage: missing `ifHighSpeed` metric, skipping metric. fullIndex=9"),
			rateMap:        interfaceRateMapWithPrevious(),
		},
		{
			name:      "missing ifHCInOctets",
			symbols:   []profiledefinition.SymbolConfig{{OID: "1.3.6.1.2.1.31.1.1.1.6", Name: "ifHCInOctets"}},
			fullIndex: "9",
			values: &valuestore.ResultValueStore{
				ColumnValues: valuestore.ColumnResultValuesType{
					// ifHCOutOctets
					"1.3.6.1.2.1.31.1.1.1.10": map[string]valuestore.ResultValue{
						"9": {
							Value: 1000000.0,
						},
					},
					// ifHighSpeed
					"1.3.6.1.2.1.31.1.1.1.15": map[string]valuestore.ResultValue{
						"9": {
							Value: 80.0,
						},
					},
				},
			},
			expectedMetric: []Metric{},
			expectedError:  fmt.Errorf("bandwidth usage: missing `ifHCInOctets` metric, skipping this row. fullIndex=9"),
			rateMap:        interfaceRateMapWithPrevious(),
		},
		{
			name:      "missing ifHCOutOctets",
			symbols:   []profiledefinition.SymbolConfig{{OID: "1.3.6.1.2.1.31.1.1.1.6", Name: "ifHCOutOctets"}},
			fullIndex: "9",
			values: &valuestore.ResultValueStore{
				ColumnValues: valuestore.ColumnResultValuesType{
					// ifHCOutOctets
					"1.3.6.1.2.1.31.1.1.1.10": map[string]valuestore.ResultValue{
						"9": {
							Value: 1000000.0,
						},
					},
					// ifHighSpeed
					"1.3.6.1.2.1.31.1.1.1.15": map[string]valuestore.ResultValue{
						"9": {
							Value: 80.0,
						},
					},
				},
			},
			expectedMetric: []Metric{},
			expectedError:  fmt.Errorf("bandwidth usage: missing `ifHCOutOctets` metric, skipping this row. fullIndex=9"),
		},
		{
			name:      "missing ifHCInOctets value",
			symbols:   []profiledefinition.SymbolConfig{{OID: "1.3.6.1.2.1.31.1.1.1.6", Name: "ifHCInOctets"}},
			fullIndex: "9",
			values: &valuestore.ResultValueStore{
				ColumnValues: valuestore.ColumnResultValuesType{
					// ifHCInOctets
					"1.3.6.1.2.1.31.1.1.1.6": map[string]valuestore.ResultValue{
						"9999": {
							Value: 5000000.0,
						},
					},
					// ifHCOutOctets
					"1.3.6.1.2.1.31.1.1.1.10": map[string]valuestore.ResultValue{
						"9": {
							Value: 1000000.0,
						},
					},
					// ifHighSpeed
					"1.3.6.1.2.1.31.1.1.1.15": map[string]valuestore.ResultValue{
						"9": {
							Value: 80.0,
						},
					},
				},
			},
			expectedMetric: []Metric{},
			expectedError:  fmt.Errorf("bandwidth usage: missing value for `ifHCInOctets` metric, skipping this row. fullIndex=9"),
		},
		{
			name:      "missing ifHighSpeed value",
			symbols:   []profiledefinition.SymbolConfig{{OID: "1.3.6.1.2.1.31.1.1.1.6", Name: "ifHCInOctets"}},
			fullIndex: "9",
			values: &valuestore.ResultValueStore{
				ColumnValues: valuestore.ColumnResultValuesType{
					// ifHCInOctets
					"1.3.6.1.2.1.31.1.1.1.6": map[string]valuestore.ResultValue{
						"9": {
							Value: 5000000.0,
						},
					},
					// ifHCOutOctets
					"1.3.6.1.2.1.31.1.1.1.10": map[string]valuestore.ResultValue{
						"9": {
							Value: 1000000.0,
						},
					},
					// ifHighSpeed
					"1.3.6.1.2.1.31.1.1.1.15": map[string]valuestore.ResultValue{
						"999": {
							Value: 80.0,
						},
					},
				},
			},
			expectedMetric: []Metric{},
			expectedError:  fmt.Errorf("bandwidth usage: missing value for `ifHighSpeed`, skipping this row. fullIndex=9"),
		},
		{
			name:      "cannot convert ifHighSpeed to float",
			symbols:   []profiledefinition.SymbolConfig{{OID: "1.3.6.1.2.1.31.1.1.1.6", Name: "ifHCInOctets"}},
			fullIndex: "9",
			values: &valuestore.ResultValueStore{
				ColumnValues: valuestore.ColumnResultValuesType{
					// ifHCInOctets
					"1.3.6.1.2.1.31.1.1.1.6": map[string]valuestore.ResultValue{
						"9": {
							Value: 5000000.0,
						},
					},
					// ifHCOutOctets
					"1.3.6.1.2.1.31.1.1.1.10": map[string]valuestore.ResultValue{
						"9": {
							Value: 1000000.0,
						},
					},
					// ifHighSpeed
					"1.3.6.1.2.1.31.1.1.1.15": map[string]valuestore.ResultValue{
						"9": {
							Value: "abc",
						},
					},
				},
			},
			expectedMetric: []Metric{},
			expectedError:  fmt.Errorf("failed to convert ifHighSpeedValue to float64: failed to parse `abc`: strconv.ParseFloat: parsing \"abc\": invalid syntax"),
		},
		{
			name:      "cannot convert ifHCInOctets to float",
			symbols:   []profiledefinition.SymbolConfig{{OID: "1.3.6.1.2.1.31.1.1.1.6", Name: "ifHCInOctets"}},
			fullIndex: "9",
			values: &valuestore.ResultValueStore{
				ColumnValues: valuestore.ColumnResultValuesType{
					// ifHCInOctets
					"1.3.6.1.2.1.31.1.1.1.6": map[string]valuestore.ResultValue{
						"9": {
							Value: "abc",
						},
					},
					// ifHCOutOctets
					"1.3.6.1.2.1.31.1.1.1.10": map[string]valuestore.ResultValue{
						"9": {
							Value: 1000000.0,
						},
					},
					// ifHighSpeed
					"1.3.6.1.2.1.31.1.1.1.15": map[string]valuestore.ResultValue{
						"9": {
							Value: 80.0,
						},
					},
				},
			},
			expectedMetric: []Metric{},
			expectedError:  fmt.Errorf("failed to convert octetsValue to float64: failed to parse `abc`: strconv.ParseFloat: parsing \"abc\": invalid syntax"),
		},
		{
			name: "[custom speed] snmp.ifBandwidthIn/OutUsage.rate with custom interface speed matched by name",
			symbols: []profiledefinition.SymbolConfig{
				{OID: "1.3.6.1.2.1.31.1.1.1.6", Name: "ifHCInOctets"},
				{OID: "1.3.6.1.2.1.31.1.1.1.10", Name: "ifHCOutOctets"},
			},
			fullIndex: "9",
			interfaceConfigs: []snmpintegration.InterfaceConfig{{
				MatchField: "name",
				MatchValue: "eth0",
				InSpeed:    160_000_000,
				OutSpeed:   40_000_000,
				Tags:       []string{"muted", "customIfTagKey:customIfTagValue"},
			}},
			tags: []string{
				"interface:eth0",
				"muted",
				"customIfTagKey:customIfTagValue",
			},
			values: &valuestore.ResultValueStore{
				ColumnValues: valuestore.ColumnResultValuesType{
					// ifHCInOctets
					"1.3.6.1.2.1.31.1.1.1.6": map[string]valuestore.ResultValue{
						"9": {
							Value: 5000000.0,
						},
					},
					// ifHCOutOctets
					"1.3.6.1.2.1.31.1.1.1.10": map[string]valuestore.ResultValue{
						"9": {
							Value: 1000000.0,
						},
					},
					// ifHighSpeed
					"1.3.6.1.2.1.31.1.1.1.15": map[string]valuestore.ResultValue{
						"9": {
							Value: 80.0,
						},
					},
				},
			},
			expectedMetric: []Metric{
				// ((5000000 * 8) / (160 * 1000000)) * 100 = 25.0
				// previous sample's usage value from map: 20
				// rate: (25 - 20) / (30 - 15)
				{"snmp.ifBandwidthInUsage.rate", 5.0 / 15.0},
				// ((1000000 * 8) / (40 * 1000000)) * 100 = 20.0
				// previous sample's usage value from map: 10
				// rate: (20 - 10) / (30 / 15)
				{"snmp.ifBandwidthOutUsage.rate", 10.0 / 15.0},
			},
			rateMap: interfaceRateMapWithConfig(),
		},
		{
			name: "[custom speed] snmp.ifBandwidthIn/OutUsage.rate with custom interface speed matched by index",
			symbols: []profiledefinition.SymbolConfig{
				{OID: "1.3.6.1.2.1.31.1.1.1.6", Name: "ifHCInOctets"},
				{OID: "1.3.6.1.2.1.31.1.1.1.10", Name: "ifHCOutOctets"},
			},
			fullIndex: "9",
			interfaceConfigs: []snmpintegration.InterfaceConfig{{
				MatchField: "index",
				MatchValue: "9",
				InSpeed:    160_000_000,
				OutSpeed:   40_000_000,
				Tags:       []string{"muted", "customIfTagKey:customIfTagValue"},
			}},
			tags: []string{
				"interface:eth0",
				"muted",
				"customIfTagKey:customIfTagValue",
			},
			values: &valuestore.ResultValueStore{
				ColumnValues: valuestore.ColumnResultValuesType{
					// ifHCInOctets
					"1.3.6.1.2.1.31.1.1.1.6": map[string]valuestore.ResultValue{
						"9": {
							Value: 5000000.0,
						},
					},
					// ifHCOutOctets
					"1.3.6.1.2.1.31.1.1.1.10": map[string]valuestore.ResultValue{
						"9": {
							Value: 1000000.0,
						},
					},
					// ifHighSpeed
					"1.3.6.1.2.1.31.1.1.1.15": map[string]valuestore.ResultValue{
						"9": {
							Value: 80.0,
						},
					},
				},
			},
			expectedMetric: []Metric{
				// ((5000000 * 8) / (160 * 1000000)) * 100 = 25.0
				// previous sample's usage value: 20
				// rate: (25 - 20) / (30 - 15)
				{"snmp.ifBandwidthInUsage.rate", 5.0 / 15.0},
				// ((1000000 * 8) / (40 * 1000000)) * 100 = 20.0
				// previous sample's usage value: 10
				// rate: (20 - 10) / (30 / 15)
				{"snmp.ifBandwidthOutUsage.rate", 10.0 / 15.0},
			},
			rateMap: interfaceRateMapWithConfig(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := mocksender.NewMockSender("testID") // required to initiate aggregator
			sender.On("Gauge", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
			sender.On("Rate", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

			TimeNow = common.MockTimeNow

			ms := &MetricSender{
				sender:                  sender,
				interfaceConfigs:        tt.interfaceConfigs,
				interfaceBandwidthState: tt.rateMap,
				deviceID:                mockDeviceID,
			}
			for _, symbol := range tt.symbols {
				err := ms.sendBandwidthUsageMetric(symbol, tt.fullIndex, tt.values, tt.tags)
				assert.Equal(t, tt.expectedError, err)
			}

			for _, metric := range tt.expectedMetric {
				sender.AssertMetric(t, "Gauge", metric.name, metric.value, "", tt.tags)
			}
		})
	}
}

func Test_metricSender_sendIfSpeedMetrics(t *testing.T) {
	type Metric struct {
		name  string
		value float64
		tags  []string
	}
	tests := []struct {
		name             string
		symbol           profiledefinition.SymbolConfig
		fullIndex        string
		values           *valuestore.ResultValueStore
		tags             []string
		interfaceConfigs []snmpintegration.InterfaceConfig
		expectedMetric   []Metric
	}{
		{
			name:      "InSpeed and OutSpeed Override",
			symbol:    profiledefinition.SymbolConfig{OID: "1.3.6.1.2.1.31.1.1.1.6", Name: "ifHCInOctets"},
			fullIndex: "9",
			interfaceConfigs: []snmpintegration.InterfaceConfig{{
				MatchField: "index",
				MatchValue: "9",
				InSpeed:    160_000_000,
				OutSpeed:   40_000_000,
			}},
			tags: []string{"interface:eth0"},
			values: &valuestore.ResultValueStore{
				ColumnValues: valuestore.ColumnResultValuesType{
					// ifHighSpeed
					"1.3.6.1.2.1.31.1.1.1.15": map[string]valuestore.ResultValue{
						"9": {
							Value: 80.0,
						},
					},
				},
			},
			expectedMetric: []Metric{
				{"snmp.ifInSpeed", 160_000_000, []string{"interface:eth0", "speed_source:custom"}},
				{"snmp.ifOutSpeed", 40_000_000, []string{"interface:eth0", "speed_source:custom"}},
			},
		},
		{
			name:      "InSpeed and OutSpeed Override with custom tags",
			symbol:    profiledefinition.SymbolConfig{OID: "1.3.6.1.2.1.31.1.1.1.6", Name: "ifHCInOctets"},
			fullIndex: "9",
			interfaceConfigs: []snmpintegration.InterfaceConfig{{
				MatchField: "index",
				MatchValue: "9",
				InSpeed:    160_000_000,
				OutSpeed:   40_000_000,
				Tags:       []string{"muted", "customKey:customValue"},
			}},
			tags: []string{"interface:eth0"},
			values: &valuestore.ResultValueStore{
				ColumnValues: valuestore.ColumnResultValuesType{
					// ifHighSpeed
					"1.3.6.1.2.1.31.1.1.1.15": map[string]valuestore.ResultValue{
						"9": {
							Value: 80.0,
						},
					},
				},
			},
			expectedMetric: []Metric{
				{"snmp.ifInSpeed", 160_000_000, []string{"interface:eth0", "speed_source:custom", "muted", "customKey:customValue"}},
				{"snmp.ifOutSpeed", 40_000_000, []string{"interface:eth0", "speed_source:custom", "muted", "customKey:customValue"}},
			},
		},
		{
			name:      "InSpeed Override but not OutSpeed Override",
			symbol:    profiledefinition.SymbolConfig{OID: "1.3.6.1.2.1.31.1.1.1.6", Name: "ifHCInOctets"},
			fullIndex: "9",
			interfaceConfigs: []snmpintegration.InterfaceConfig{{
				MatchField: "index",
				MatchValue: "9",
				InSpeed:    160_000_000,
			}},
			tags: []string{"interface:eth0"},
			values: &valuestore.ResultValueStore{
				ColumnValues: valuestore.ColumnResultValuesType{
					// ifHighSpeed
					"1.3.6.1.2.1.31.1.1.1.15": map[string]valuestore.ResultValue{
						"9": {
							Value: 80.0,
						},
					},
				},
			},
			expectedMetric: []Metric{
				{"snmp.ifInSpeed", 160_000_000, []string{"interface:eth0", "speed_source:custom"}},
				{"snmp.ifOutSpeed", 80_000_000, []string{"interface:eth0", "speed_source:device"}},
			},
		},
		{
			name:      "InSpeed and OutSpeed config with zero values",
			symbol:    profiledefinition.SymbolConfig{OID: "1.3.6.1.2.1.31.1.1.1.6", Name: "ifHCInOctets"},
			fullIndex: "9",
			interfaceConfigs: []snmpintegration.InterfaceConfig{{
				MatchField: "index",
				MatchValue: "9",
				InSpeed:    0,
				OutSpeed:   0,
			}},
			values: &valuestore.ResultValueStore{
				ColumnValues: valuestore.ColumnResultValuesType{
					// ifHighSpeed
					"1.3.6.1.2.1.31.1.1.1.15": map[string]valuestore.ResultValue{
						"9": {
							Value: 80.0,
						},
					},
				},
			},
			expectedMetric: []Metric{
				{"snmp.ifInSpeed", 80_000_000, []string{"speed_source:device"}},
				{"snmp.ifOutSpeed", 80_000_000, []string{"speed_source:device"}},
			},
		},
		{
			name:             "no interface config found",
			symbol:           profiledefinition.SymbolConfig{OID: "1.3.6.1.2.1.31.1.1.1.6", Name: "ifHCInOctets"},
			fullIndex:        "9",
			interfaceConfigs: []snmpintegration.InterfaceConfig{},
			values: &valuestore.ResultValueStore{
				ColumnValues: valuestore.ColumnResultValuesType{
					// ifHighSpeed
					"1.3.6.1.2.1.31.1.1.1.15": map[string]valuestore.ResultValue{
						"9": {
							Value: 80.0,
						},
					},
				},
			},
			expectedMetric: []Metric{
				{"snmp.ifInSpeed", 80_000_000, []string{"speed_source:device"}},
				{"snmp.ifOutSpeed", 80_000_000, []string{"speed_source:device"}},
			},
		},
		{
			name:             "no interface config found and no ifHighSpeed",
			symbol:           profiledefinition.SymbolConfig{OID: "1.3.6.1.2.1.31.1.1.1.6", Name: "ifHCInOctets"},
			fullIndex:        "9",
			interfaceConfigs: []snmpintegration.InterfaceConfig{},
			values:           &valuestore.ResultValueStore{},
			expectedMetric:   []Metric{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := mocksender.NewMockSender("testID") // required to initiate aggregator
			sender.On("Gauge", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

			ms := &MetricSender{
				sender:                  sender,
				interfaceConfigs:        tt.interfaceConfigs,
				interfaceBandwidthState: NewInterfaceBandwidthState(),
			}
			ms.sendIfSpeedMetrics(tt.symbol, tt.fullIndex, tt.values, tt.tags)

			for _, metric := range tt.expectedMetric {
				sender.AssertMetric(t, "Gauge", metric.name, metric.value, "", metric.tags)
			}
			assert.Equal(t, len(tt.expectedMetric), len(sender.Mock.Calls))
		})
	}
}

func Test_metricSender_sendInterfaceVolumeMetrics(t *testing.T) {
	type Metric struct {
		metricMethod string
		name         string
		value        float64
	}
	tests := []struct {
		name           string
		symbol         profiledefinition.SymbolConfig
		fullIndex      string
		values         *valuestore.ResultValueStore
		expectedMetric []Metric
	}{
		{
			"snmp.ifBandwidthInUsage.Rate submitted",
			profiledefinition.SymbolConfig{OID: "1.3.6.1.2.1.31.1.1.1.6", Name: "ifHCInOctets"},
			"9",
			&valuestore.ResultValueStore{
				ColumnValues: valuestore.ColumnResultValuesType{
					// ifHCInOctets
					"1.3.6.1.2.1.31.1.1.1.6": map[string]valuestore.ResultValue{
						"9": {
							Value: 5000000.0,
						},
					},
					// ifHCOutOctets
					"1.3.6.1.2.1.31.1.1.1.10": map[string]valuestore.ResultValue{
						"9": {
							Value: 1000000.0,
						},
					},
					// ifHighSpeed
					"1.3.6.1.2.1.31.1.1.1.15": map[string]valuestore.ResultValue{
						"9": {
							Value: 80.0,
						},
					},
				},
			},
			[]Metric{
				{"Gauge", "snmp.ifBandwidthInUsage.rate", 20.0 / 15.0},
				{"Gauge", "snmp.ifInSpeed", 80_000_000},
				{"Gauge", "snmp.ifOutSpeed", 80_000_000},
			},
		},
		{
			"should complete even on error",
			profiledefinition.SymbolConfig{OID: "1.3.6.1.2.1.31.1.1.1.6", Name: "ifHCInOctets"},
			"9",
			&valuestore.ResultValueStore{
				ColumnValues: valuestore.ColumnResultValuesType{
					// ifHCInOctets
					"1.3.6.1.2.1.31.1.1.1.6": map[string]valuestore.ResultValue{
						"9": {
							Value: 5000000.0,
						},
					},
					// ifHCOutOctets
					"1.3.6.1.2.1.31.1.1.1.10": map[string]valuestore.ResultValue{
						"9": {
							Value: 1000000.0,
						},
					},
					// ifHighSpeed
					"1.3.6.1.2.1.31.1.1.1.15": map[string]valuestore.ResultValue{
						"999": {
							Value: 80.0,
						},
					},
				},
			},
			[]Metric{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := mocksender.NewMockSender("testID") // required to initiate aggregator
			sender.On("Rate", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
			sender.On("Gauge", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

			TimeNow = common.MockTimeNow
			ms := &MetricSender{
				sender:                  sender,
				interfaceBandwidthState: interfaceRateMapWithPrevious(),
				deviceID:                mockDeviceID,
			}
			tags := []string{"foo:bar"}
			ms.sendInterfaceVolumeMetrics(tt.symbol, tt.fullIndex, tt.values, tags)

			for _, metric := range tt.expectedMetric {
				sender.AssertMetric(t, metric.metricMethod, metric.name, metric.value, "", tags)
			}
		})
	}
}
