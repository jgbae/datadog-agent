// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux_bpf

package usm

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	nethttp "net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/cihub/seelog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	manager "github.com/DataDog/ebpf-manager"

	"github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/ebpf/ebpftest"
	networkconfig "github.com/DataDog/datadog-agent/pkg/network/config"
	netlink "github.com/DataDog/datadog-agent/pkg/network/netlink/testutil"
	"github.com/DataDog/datadog-agent/pkg/network/protocols"
	"github.com/DataDog/datadog-agent/pkg/network/protocols/http"
	"github.com/DataDog/datadog-agent/pkg/network/protocols/http/testutil"
	usmhttp2 "github.com/DataDog/datadog-agent/pkg/network/protocols/http2"
	libtelemetry "github.com/DataDog/datadog-agent/pkg/network/protocols/telemetry"
	"github.com/DataDog/datadog-agent/pkg/util/kernel"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

func TestMain(m *testing.M) {
	logLevel := os.Getenv("DD_LOG_LEVEL")
	if logLevel == "" {
		logLevel = "warn"
	}
	log.SetupLogger(seelog.Default, logLevel)
	os.Exit(m.Run())
}

const (
	kb = 1024
	mb = 1024 * kb

	http2SrvAddr    = "http://127.0.0.1:8082"
	http2SrvPortStr = ":8082"
	http2SrvPort    = 8082
)

var (
	emptyBody = []byte(nil)
	kv        = kernel.MustHostVersion()
)

func skipIfUSMNotSupported(t *testing.T) {
	if kv < http.MinimumKernelVersion {
		t.Skipf("USM is not supported on %v", kv)
	}
}

func TestMonitorProtocolFail(t *testing.T) {
	failingStartupMock := func(_ *manager.Manager) error {
		return fmt.Errorf("mock error")
	}

	testCases := []struct {
		name string
		spec protocolMockSpec
	}{
		{name: "PreStart fails", spec: protocolMockSpec{preStartFn: failingStartupMock}},
		{name: "PostStart fails", spec: protocolMockSpec{postStartFn: failingStartupMock}},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// Replace the HTTP protocol with a Mock
			patchProtocolMock(t, tt.spec)

			cfg := networkconfig.New()
			cfg.EnableHTTPMonitoring = true
			monitor, err := NewMonitor(cfg, nil, nil, nil)
			skipIfNotSupported(t, err)
			require.NoError(t, err)
			t.Cleanup(monitor.Stop)

			err = monitor.Start()
			require.ErrorIs(t, err, errNoProtocols)
		})
	}
}

type HTTPTestSuite struct {
	suite.Suite
}

func TestHTTP(t *testing.T) {
	skipIfUSMNotSupported(t)
	ebpftest.TestBuildModes(t, []ebpftest.BuildMode{ebpftest.Prebuilt, ebpftest.RuntimeCompiled, ebpftest.CORE}, "", func(t *testing.T) {
		suite.Run(t, new(HTTPTestSuite))
	})
}

func (s *HTTPTestSuite) TestHTTPStats() {
	t := s.T()
	t.Run("status code", func(t *testing.T) {
		testHTTPStats(t, true)
	})
	t.Run("status class", func(t *testing.T) {
		testHTTPStats(t, false)
	})
}

func testHTTPStats(t *testing.T, aggregateByStatusCode bool) {
	// Start an HTTP server on localhost:8080
	serverAddr := "127.0.0.1:8080"
	srvDoneFn := testutil.HTTPServer(t, serverAddr, testutil.Options{
		EnableKeepAlive: true,
	})
	t.Cleanup(srvDoneFn)

	cfg := networkconfig.New()
	cfg.EnableHTTPStatsByStatusCode = aggregateByStatusCode
	monitor := newHTTPMonitorWithCfg(t, cfg)

	resp, err := nethttp.Get(fmt.Sprintf("http://%s/%d/test", serverAddr, nethttp.StatusNoContent))
	require.NoError(t, err)
	_ = resp.Body.Close()
	srvDoneFn()

	// Iterate through active connections until we find connection created above
	require.Eventuallyf(t, func() bool {
		stats := getHttpStats(t, monitor)

		for key, reqStats := range stats {
			if key.Method == http.MethodGet && strings.HasSuffix(key.Path.Content.Get(), "/test") && (key.SrcPort == 8080 || key.DstPort == 8080) {
				currentStats := reqStats.Data[reqStats.NormalizeStatusCode(204)]
				if currentStats != nil && currentStats.Count == 1 {
					return true
				}
			}
		}

		return false
	}, 3*time.Second, 100*time.Millisecond, "couldn't find http connection matching: %s", serverAddr)
}

func (s *HTTPTestSuite) TestHTTPMonitorCaptureRequestMultipleTimes() {
	t := s.T()

	for _, TCPTimestamp := range []struct {
		name  string
		value bool
	}{
		{name: "without TCP timestamp option", value: false},
		{name: "with TCP timestamp option", value: true},
	} {
		t.Run(TCPTimestamp.name, func(t *testing.T) {

			monitor := newHTTPMonitor(t)

			serverAddr := "localhost:8081"
			srvDoneFn := testutil.HTTPServer(t, serverAddr, testutil.Options{
				EnableTCPTimestamp: &TCPTimestamp.value,
			})

			client := nethttp.Client{}

			req, err := nethttp.NewRequest(httpMethods[0], fmt.Sprintf("http://%s/%d/request", serverAddr, nethttp.StatusOK), nil)
			require.NoError(t, err)

			expectedOccurrences := 10
			for i := 0; i < expectedOccurrences; i++ {
				resp, err := client.Do(req)
				require.NoError(t, err)
				// Have to read the response body to ensure the client will be able to properly close the connection.
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
			srvDoneFn()

			occurrences := 0
			require.Eventually(t, func() bool {
				stats := getHttpStats(t, monitor)
				occurrences += countRequestOccurrences(stats, req)
				return occurrences == expectedOccurrences
			}, time.Second*3, time.Millisecond*100, "Expected to find a request %d times, instead captured %d", occurrences, expectedOccurrences)
		})
	}
}

// TestHTTPMonitorLoadWithIncompleteBuffers sends thousands of requests without getting responses for them, in parallel
// we send another request. We expect to capture the another request but not the incomplete requests.
func (s *HTTPTestSuite) TestHTTPMonitorLoadWithIncompleteBuffers() {
	t := s.T()

	slowServerAddr := "localhost:8080"
	fastServerAddr := "localhost:8081"

	monitor := newHTTPMonitor(t)
	slowSrvDoneFn := testutil.HTTPServer(t, slowServerAddr, testutil.Options{
		SlowResponse: time.Millisecond * 500, // Half a second.
		WriteTimeout: time.Millisecond * 200,
		ReadTimeout:  time.Millisecond * 200,
	})

	fastSrvDoneFn := testutil.HTTPServer(t, fastServerAddr, testutil.Options{})
	abortedRequestFn := requestGenerator(t, fmt.Sprintf("%s/ignore", slowServerAddr), emptyBody)
	wg := sync.WaitGroup{}
	abortedRequests := make(chan *nethttp.Request, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := abortedRequestFn()
			abortedRequests <- req
		}()
	}
	fastReq := requestGenerator(t, fastServerAddr, emptyBody)()
	wg.Wait()
	close(abortedRequests)
	slowSrvDoneFn()
	fastSrvDoneFn()

	foundFastReq := false
	// We are iterating for a couple of iterations and making sure the aborted requests will never be found.
	// Since the every call for monitor.GetHTTPStats will delete the pop all entries, and we want to find fastReq
	// then we are using a variable to check if "we ever found it" among the iterations.
	for i := 0; i < 10; i++ {
		time.Sleep(10 * time.Millisecond)
		stats := getHttpStats(t, monitor)
		for req := range abortedRequests {
			requestNotIncluded(t, stats, req)
		}

		included, err := isRequestIncludedOnce(stats, fastReq)
		require.NoError(t, err)
		foundFastReq = foundFastReq || included
	}

	require.True(t, foundFastReq)
}

func (s *HTTPTestSuite) TestHTTPMonitorIntegrationWithResponseBody() {
	t := s.T()
	targetAddr := "localhost:8080"
	serverAddr := "localhost:8080"

	tests := []struct {
		name            string
		requestBodySize int
	}{
		{
			name:            "no body",
			requestBodySize: 0,
		},
		{
			name:            "1kb body",
			requestBodySize: 1 * kb,
		},
		{
			name:            "10kb body",
			requestBodySize: 10 * kb,
		},
		{
			name:            "100kb body",
			requestBodySize: 100 * kb,
		},
		{
			name:            "500kb body",
			requestBodySize: 500 * kb,
		},
		{
			name:            "2mb body",
			requestBodySize: 2 * mb,
		},
		{
			name:            "10mb body",
			requestBodySize: 10 * mb,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitor := newHTTPMonitor(t)
			srvDoneFn := testutil.HTTPServer(t, serverAddr, testutil.Options{
				EnableKeepAlive: true,
			})
			t.Cleanup(srvDoneFn)

			requestFn := requestGenerator(t, targetAddr, bytes.Repeat([]byte("a"), tt.requestBodySize))
			var requests []*nethttp.Request
			for i := 0; i < 100; i++ {
				requests = append(requests, requestFn())
			}
			srvDoneFn()

			assertAllRequestsExists(t, monitor, requests)
		})
	}
}

func (s *HTTPTestSuite) TestHTTPMonitorIntegrationSlowResponse() {
	t := s.T()
	targetAddr := "localhost:8080"
	serverAddr := "localhost:8080"

	tests := []struct {
		name                         string
		mapCleanerIntervalSeconds    int
		httpIdleConnectionTTLSeconds int
		slowResponseTime             int
		shouldCapture                bool
	}{
		{
			name:                         "response reaching after cleanup",
			mapCleanerIntervalSeconds:    1,
			httpIdleConnectionTTLSeconds: 1,
			slowResponseTime:             3,
			shouldCapture:                false,
		},
		{
			name:                         "response reaching before cleanup",
			mapCleanerIntervalSeconds:    1,
			httpIdleConnectionTTLSeconds: 3,
			slowResponseTime:             1,
			shouldCapture:                true,
		},
		{
			name:                         "slow response reaching after ttl but cleaner not running",
			mapCleanerIntervalSeconds:    3,
			httpIdleConnectionTTLSeconds: 1,
			slowResponseTime:             2,
			shouldCapture:                true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.ResetSystemProbeConfig(t)
			t.Setenv("DD_SERVICE_MONITORING_CONFIG_HTTP_MAP_CLEANER_INTERVAL_IN_S", strconv.Itoa(tt.mapCleanerIntervalSeconds))
			t.Setenv("DD_SERVICE_MONITORING_CONFIG_HTTP_IDLE_CONNECTION_TTL_IN_S", strconv.Itoa(tt.httpIdleConnectionTTLSeconds))
			monitor := newHTTPMonitor(t)

			slowResponseTimeout := time.Duration(tt.slowResponseTime) * time.Second
			serverTimeout := slowResponseTimeout + time.Second
			srvDoneFn := testutil.HTTPServer(t, serverAddr, testutil.Options{
				WriteTimeout: serverTimeout,
				ReadTimeout:  serverTimeout,
				SlowResponse: slowResponseTimeout,
			})
			t.Cleanup(srvDoneFn)

			// Perform a number of random requests
			req := requestGenerator(t, targetAddr, emptyBody)()
			srvDoneFn()

			// Ensure all captured transactions get sent to user-space
			time.Sleep(10 * time.Millisecond)
			stats := getHttpStats(t, monitor)

			if tt.shouldCapture {
				includesRequest(t, stats, req)
			} else {
				requestNotIncluded(t, stats, req)
			}
		})
	}
}

func (s *HTTPTestSuite) TestHTTPMonitorIntegration() {
	t := s.T()
	targetAddr := "localhost:8080"
	serverAddr := "localhost:8080"

	t.Run("with keep-alives", func(t *testing.T) {
		testHTTPMonitor(t, targetAddr, serverAddr, 100, testutil.Options{
			EnableKeepAlive: true,
		})
	})
	t.Run("without keep-alives", func(t *testing.T) {
		testHTTPMonitor(t, targetAddr, serverAddr, 100, testutil.Options{
			EnableKeepAlive: false,
		})
	})
}

func (s *HTTPTestSuite) TestHTTPMonitorIntegrationWithNAT() {
	t := s.T()
	// SetupDNAT sets up a NAT translation from 2.2.2.2 to 1.1.1.1
	netlink.SetupDNAT(t)

	targetAddr := "2.2.2.2:8080"
	serverAddr := "1.1.1.1:8080"

	t.Run("with keep-alives", func(t *testing.T) {
		testHTTPMonitor(t, targetAddr, serverAddr, 100, testutil.Options{
			EnableKeepAlive: true,
		})
	})
	t.Run("without keep-alives", func(t *testing.T) {
		testHTTPMonitor(t, targetAddr, serverAddr, 100, testutil.Options{
			EnableKeepAlive: false,
		})
	})
}

func (s *HTTPTestSuite) TestRSTPacketRegression() {
	t := s.T()

	monitor := newHTTPMonitor(t)

	serverAddr := "127.0.0.1:8080"
	srvDoneFn := testutil.HTTPServer(t, serverAddr, testutil.Options{
		EnableKeepAlive: true,
	})
	t.Cleanup(srvDoneFn)

	// Create a "raw" TCP socket that will serve as our HTTP client
	// We do this in order to configure the socket option SO_LINGER
	// so we can force a RST packet to be sent during termination
	c, err := net.DialTimeout("tcp", serverAddr, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	// Issue HTTP request
	c.Write([]byte("GET /200/foobar HTTP/1.1\nHost: 127.0.0.1:8080\n\n"))
	io.Copy(io.Discard, c)

	// Configure SO_LINGER to 0 so that triggers an RST when the socket is terminated
	require.NoError(t, c.(*net.TCPConn).SetLinger(0))
	c.Close()
	time.Sleep(100 * time.Millisecond)

	// Assert that the HTTP request was correctly handled despite its forceful termination
	stats := getHttpStats(t, monitor)
	url, err := url.Parse("http://127.0.0.1:8080/200/foobar")
	require.NoError(t, err)
	includesRequest(t, stats, &nethttp.Request{URL: url})
}

func (s *HTTPTestSuite) TestKeepAliveWithIncompleteResponseRegression() {
	t := s.T()

	monitor := newHTTPMonitor(t)

	const req = "GET /200/foobar HTTP/1.1\n"
	const rsp = "HTTP/1.1 200 OK\n"
	const serverAddr = "127.0.0.1:8080"

	srvFn := func(c net.Conn) {
		// emulates a half-transaction (beginning with a response)
		n, err := c.Write([]byte(rsp))
		require.NoError(t, err)
		require.Equal(t, len(rsp), n)

		// now we read the request from the client on the same connection
		b := make([]byte, len(req))
		n, err = c.Read(b)
		require.NoError(t, err)
		require.Equal(t, len(req), n)
		require.Equal(t, string(b), req)

		// and finally send the response completing a full HTTP transaction
		n, err = c.Write([]byte(rsp))
		require.NoError(t, err)
		require.Equal(t, len(rsp), n)
		c.Close()
	}
	srv := testutil.NewTCPServer(serverAddr, srvFn)
	done := make(chan struct{})
	srv.Run(done)
	t.Cleanup(func() { close(done) })

	c, err := net.DialTimeout("tcp", serverAddr, 5*time.Second)
	require.NoError(t, err)

	// ensure we're beginning the connection with a "headless" response from the
	// server. this emulates the case where system-probe started in the middle of
	// request/response cyle
	b := make([]byte, len(rsp))
	n, err := c.Read(b)
	require.NoError(t, err)
	require.Equal(t, len(rsp), n)
	require.Equal(t, string(b), rsp)

	// now perform a request
	n, err = c.Write([]byte(req))
	require.NoError(t, err)
	require.Equal(t, len(req), n)

	// and read the response completing a full transaction
	n, err = c.Read(b)
	require.NoError(t, err)
	require.Equal(t, len(rsp), n)
	require.Equal(t, string(b), rsp)

	// after this response, request, response cycle we should ensure that
	// we got a full HTTP transaction
	url, err := url.Parse("http://127.0.0.1:8080/200/foobar")
	require.NoError(t, err)
	assertAllRequestsExists(t, monitor, []*nethttp.Request{{URL: url, Method: "GET"}})
}

type USMHTTP2Suite struct {
	suite.Suite
}

type captureRange struct {
	lower int
	upper int
}

func TestHTTP2(t *testing.T) {
	currKernelVersion, err := kernel.HostVersion()
	require.NoError(t, err)
	if currKernelVersion < usmhttp2.MinimumKernelVersion {
		t.Skipf("HTTP2 monitoring can not run on kernel before %v", usmhttp2.MinimumKernelVersion)
	}

	ebpftest.TestBuildModes(t, []ebpftest.BuildMode{ebpftest.Prebuilt, ebpftest.RuntimeCompiled, ebpftest.CORE}, "", func(t *testing.T) {
		suite.Run(t, new(USMHTTP2Suite))
	})
}

func (s *USMHTTP2Suite) TestHTTP2DynamicTableCleanup() {
	t := s.T()
	cfg := networkconfig.New()
	cfg.EnableHTTP2Monitoring = true
	cfg.HTTP2DynamicTableMapCleanerInterval = 5 * time.Second

	startH2CServer(t)

	monitor, err := NewMonitor(cfg, nil, nil, nil)
	require.NoError(t, err)
	require.NoError(t, monitor.Start())
	defer monitor.Stop()

	numberOfRequests := usmhttp2.HTTP2TerminatedBatchSize
	clients := getClientsArray(t, 2)
	for i := 0; i < numberOfRequests; i++ {
		req, err := clients[i%2].Post(fmt.Sprintf("%s/test-%d", http2SrvAddr, i+1), "application/json", bytes.NewReader([]byte("test")))
		require.NoError(t, err, "could not make request")
		req.Body.Close()
	}

	matches := PrintableInt(0)

	require.Eventuallyf(t, func() bool {
		stats := monitor.GetProtocolStats()
		http2Stats, ok := stats[protocols.HTTP2]
		if !ok {
			return false
		}
		http2StatsTyped := http2Stats.(map[http.Key]*http.RequestStats)
		for key, stat := range http2StatsTyped {
			if (key.DstPort == http2SrvPort || key.SrcPort == http2SrvPort) && key.Method == http.MethodPost && strings.HasPrefix(key.Path.Content.Get(), "/test") {
				matches.Add(stat.Data[200].Count)
			}
		}

		return matches.Load() == numberOfRequests
	}, time.Second*10, time.Millisecond*100, "%v != %v", &matches, numberOfRequests)

	for _, client := range clients {
		client.CloseIdleConnections()
	}

	dynamicTableMap, _, err := monitor.ebpfProgram.GetMap("http2_dynamic_table")
	require.NoError(t, err)
	iterator := dynamicTableMap.Iterate()
	key := make([]byte, dynamicTableMap.KeySize())
	value := make([]byte, dynamicTableMap.ValueSize())
	count := 0
	for iterator.Next(&key, &value) {
		count++
	}
	require.GreaterOrEqual(t, count, 0)

	require.Eventually(t, func() bool {
		iterator = dynamicTableMap.Iterate()
		count = 0
		for iterator.Next(&key, &value) {
			count++
		}

		return count == 0
	}, cfg.HTTP2DynamicTableMapCleanerInterval*4, time.Millisecond*100)
}

func (s *USMHTTP2Suite) TestHTTP2ManyDifferentPaths() {
	t := s.T()
	cfg := networkconfig.New()
	cfg.EnableHTTP2Monitoring = true

	startH2CServer(t)

	monitor, err := NewMonitor(cfg, nil, nil, nil)
	require.NoError(t, err)
	require.NoError(t, monitor.Start())
	defer monitor.Stop()

	// Should be bigger than the length of the http2_dynamic_table which is 1024
	const numberOfRequests = 1500
	const expectedNumberOfRequests = numberOfRequests * 2
	clients := getClientsArray(t, 1)
	for i := 0; i < numberOfRequests; i++ {
		for j := 0; j < 2; j++ {
			req, err := clients[0].Post(fmt.Sprintf("%s/test-%d", http2SrvAddr, i+1), "application/json", bytes.NewReader([]byte("test")))
			require.NoError(t, err, "could not make request")
			req.Body.Close()
		}
	}

	matches := PrintableInt(0)

	seenRequests := map[string]int{}
	assert.Eventuallyf(t, func() bool {
		stats := monitor.GetProtocolStats()
		http2Stats, ok := stats[protocols.HTTP2]
		if !ok {
			return false
		}
		http2StatsTyped := http2Stats.(map[http.Key]*http.RequestStats)
		for key, stat := range http2StatsTyped {
			if (key.DstPort == http2SrvPort || key.SrcPort == http2SrvPort) && key.Method == http.MethodPost && strings.HasPrefix(key.Path.Content.Get(), "/test") {
				if _, ok := seenRequests[key.Path.Content.Get()]; !ok {
					seenRequests[key.Path.Content.Get()] = 0
				}
				seenRequests[key.Path.Content.Get()] += stat.Data[200].Count
				matches.Add(stat.Data[200].Count)
			}
		}

		// Due to a known issue in http2, we might consider an RST packet as a response to a request and therefore
		// we might capture a request twice. This is why we are expecting to see 2*numberOfRequests instead of
		return expectedNumberOfRequests <= matches.Load() && matches.Load() <= expectedNumberOfRequests+1
	}, time.Second*10, time.Millisecond*100, "%v != %v", &matches, 2*numberOfRequests)

	for i := 0; i < numberOfRequests; i++ {
		if v, ok := seenRequests[fmt.Sprintf("/test-%d", i+1)]; !ok || v != 2 {
			t.Logf("path: /test-%d should have 2 occurrences but instead has %d", i+1, v)
		}
	}
}

func (s *USMHTTP2Suite) TestSimpleHTTP2() {
	t := s.T()
	cfg := networkconfig.New()
	cfg.EnableHTTP2Monitoring = true

	startH2CServer(t)

	tests := []struct {
		name              string
		runClients        func(t *testing.T, clientsCount int)
		expectedEndpoints map[http.Key]captureRange
	}{
		{
			name: " / path",
			runClients: func(t *testing.T, clientsCount int) {
				clients := getClientsArray(t, clientsCount)

				for i := 0; i < 1000; i++ {
					client := clients[getClientsIndex(i, clientsCount)]
					req, err := client.Post(http2SrvAddr+"/", "application/json", bytes.NewReader([]byte("test")))
					require.NoError(t, err, "could not make request")
					req.Body.Close()
				}
			},
			expectedEndpoints: map[http.Key]captureRange{
				{
					Path:   http.Path{Content: http.Interner.GetString("/")},
					Method: http.MethodPost,
				}: {
					lower: 999,
					upper: 1001,
				},
			},
		},
		{
			name: " /index.html path",
			runClients: func(t *testing.T, clientsCount int) {
				clients := getClientsArray(t, clientsCount)

				for i := 0; i < 1000; i++ {
					client := clients[getClientsIndex(i, clientsCount)]
					req, err := client.Post(http2SrvAddr+"/index.html", "application/json", bytes.NewReader([]byte("test")))
					require.NoError(t, err, "could not make request")
					req.Body.Close()
				}
			},
			expectedEndpoints: map[http.Key]captureRange{
				{
					Path:   http.Path{Content: http.Interner.GetString("/index.html")},
					Method: http.MethodPost,
				}: {
					lower: 999,
					upper: 1001,
				},
			},
		},
	}
	for _, tt := range tests {
		for _, clientCount := range []int{1, 2, 5} {
			testNameSuffix := fmt.Sprintf("-different clients - %v", clientCount)
			t.Run(tt.name+testNameSuffix, func(t *testing.T) {
				monitor, err := NewMonitor(cfg, nil, nil, nil)
				require.NoError(t, err)
				require.NoError(t, monitor.Start())
				defer monitor.Stop()

				tt.runClients(t, clientCount)

				res := make(map[http.Key]int)
				assert.Eventually(t, func() bool {
					stats := monitor.GetProtocolStats()
					http2Stats, ok := stats[protocols.HTTP2]
					if !ok {
						return false
					}
					http2StatsTyped := http2Stats.(map[http.Key]*http.RequestStats)
					for key, stat := range http2StatsTyped {
						if key.DstPort == http2SrvPort || key.SrcPort == http2SrvPort {
							count := stat.Data[200].Count
							newKey := http.Key{
								Path:   http.Path{Content: key.Path.Content},
								Method: key.Method,
							}
							if _, ok := res[newKey]; !ok {
								res[newKey] = count
							} else {
								res[newKey] += count
							}
						}
					}

					if len(res) != len(tt.expectedEndpoints) {
						return false
					}

					for key, count := range res {
						valRange, ok := tt.expectedEndpoints[key]
						if !ok {
							return false
						}
						if count < valRange.lower || count > valRange.upper {
							return false
						}
					}

					return true
				}, time.Second*5, time.Millisecond*100, "%v != %v", res, tt.expectedEndpoints)
				if t.Failed() {
					o, err := monitor.DumpMaps("http2_in_flight")
					if err != nil {
						t.Logf("failed dumping http2_in_flight: %s", err)
					} else {
						t.Log(o)
					}
				}
			})
		}
	}
}

var http2UniquePaths = []string{
	// size 18 bucket 0
	"C9ZaSMOpthT9XaRh9yc6AKqfIjT43M8gOz3p9ASKCNRIcLbc3PTqEoms2SDwt6Q90QM7DxjWKlmZUfRU1eOx5DjQOOhLaIJQke4N",
	// size 122 bucket 1
	"ZtZuUQeVB7BOl3F45oFicOOOJl21ePFwunMBvBh3bXPMBZqdEZepVsemYA0frZb5M83VHLDWq68KFELDHu0Xo28lzpzO3L7kDXuYuClivgEgURUn47kfwfUfW1PKjfsV6HaYpAZxly48lTGiRIXRINVC8b9jbbjNA6nBzubenIctRKBDywbifoO0YVzCSLm3qjpc4U2k4CZAoUm2X3mGCTwPawlEt8Z2KgFjcB",
	// size 131, bucket 2
	"RDBVk5COXAz52GzvuHVWRawNoKhmfxhBiTuyj5QZ6qR1DMsNOn4sWFLnaGXVzrqA8NLr2CaW1IDupzh9AzJlIvgYSf6OYIafIOsImL5O9M3AHzUHGMJ0KhjYGJAzXeTgvwl2qYWmlD9UYGELFpBSzJpykoriocvl3RRoYt4lQ0favN3ptHYNUPJ1XDjH7CdRoad2jrIcKd04J49ZFTHJK7l6MyUe6XWEcfMB3sI2h1jY9z2z",
	// size 145, bucket 3
	"T5r8QcP8qCiKVwhWlaxWjYCX8IrTmPrt2HRjfQJP2PxbWjLm8dP4BTDxUAmXJJWNyv4HIIaR3Fj6n8Tu6vSoDcBtKFuMqIPAdYEJt0qo2aaYDKomIJv74z7SiN96GrOufPTm6Eutl3JGeAKW2b0dZ4VYUsIOO8aheEOGmyhyWBymgCtBcXeki1HZZz9J65QColhuslBmbLTfg5sIFA9jImyF4CANIn53lhxzi4hslSCHaGz8JrCqnfbueayY21jUGwCF",
	// size 155, bucket 4
	"VP4zOrIPiGhLDLSJYSVU78yUcb8CkU0dVDIZqPq98gVoenX5p1zS6cRX4LtrfSYKCQFX6MquluhDD2GPjZYFIraDLIHCno3yipQBLPGcPbPTgv9SD6jOlHMuLjmsGxyC3y2Hk61bWA6Af4D2SYS0q3BS7ahJ0vjddYYBRIpwMOOIez2jaR56rPcGCRW2eq0T1xAvaNqOpjSuDVYAN7miDxAqFn69eIfJm6YTKgEFy3cua5MBCn7xFxcFvsC11x9D9TzqHZDxGd1SoI",
	// size 163, bucket 5
	"X2YRUwfeNEmYWkk0bACThVya8MoSUkR7ZKANCPYkIGHvF9CWGA0rxXKsGogQag7HsJfmgaar3TiOTRUb3ynbmiOz3As9rXYjRGNRdCWGgdBPL8nGa6WheGlJLNtIVsUcxSerNQKmoQqqDLjGftbKXjqdMJLVY6UyECeXOKrrFU9aHx2fjlk2qMNDUptYWuzPPCWAnKOV7PhkkDiXWySbEzMTRb1LsgYxdpivawvxWbs39T9G3itElybS10z6m2MtyhNxjZqrxQnOKuAQrFs6vh9u",
	// size 175, bucket 6
	"bq5bcpUgiW1CpKgwdRVIulFMkwRenJWYdW8aek69anIV8w3br0pjGNtfnoPCyj4HUMD5MxWB2xM4XGp7fZ1JRHvskRZEgmoM7ag9BeuigmH05p7dzMwKsD76MqKyPmfhwBUZHLKtJ52ia3mOuMvyYiQNwA6KAU509bwuy4NCREVUAP76WFeAzr0jBvqMFXLg3eQQERIW0tKTcjQg8m9Jx49Vo679QcT40e1R1cjvIauX8uJ2PDQiyiVm2r81mSaabT7pARQcY6HN3pDgBsbpL3E7J7LIWD1YiDr28pfu8mAs",
	// size 183, bucket 7
	"LUhWUWPMztVFuEs83i7RmoxRiV1KzOq0NsZmGXVyW49BbBaL63m8H5vDwiewrrKbldXBuctplDxB28QekDclM6cO9BIsRqvzS3a802aOkRHTEruotA8Xh5K9GOMv9DzdoOL9P3GFPsUPgBy0mzFyyRJGk3JXpIH290Bj2FIRnIIpIjjKE1akeaimsuGEheA4D95axRpGmz4cm2s74UiksfBi4JnVX2cBzZN3oQaMt7zrWofwyzcZeF5W1n6BAQWxPPWe4Jyoc34jQ2fiEXQO0NnXe1RFbBD1E33a0OycziXZH9hEP23xvh",
}

func (s *USMHTTP2Suite) TestHTTP2KernelTelemetry() {
	t := s.T()
	cfg := networkconfig.New()
	cfg.EnableHTTP2Monitoring = true

	startH2CServer(t)

	tests := []struct {
		name              string
		runClients        func(t *testing.T, clientsCount int)
		expectedTelemetry *usmhttp2.HTTP2Telemetry
	}{
		{
			name: "Fill each bucket",
			runClients: func(t *testing.T, clientsCount int) {
				clients := getClientsArray(t, clientsCount)
				for _, path := range http2UniquePaths {
					client := clients[getClientsIndex(1, clientsCount)]
					req, err := client.Post(http2SrvAddr+"/"+path, "application/json", bytes.NewReader([]byte("test")))
					require.NoError(t, err, "could not make request")
					req.Body.Close()
				}
			},

			expectedTelemetry: &usmhttp2.HTTP2Telemetry{
				Request_seen:                     8,
				Response_seen:                    8,
				End_of_stream:                    16,
				End_of_stream_rst:                0,
				Path_exceeds_frame:               0,
				Exceeding_max_interesting_frames: 0,
				Exceeding_max_frames_to_filter:   0,
				Path_size_bucket:                 [8]uint64{1, 1, 1, 1, 1, 1, 1, 1},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitor, err := NewMonitor(cfg, nil, nil, nil)
			require.NoError(t, err)
			require.NoError(t, monitor.Start())
			defer monitor.Stop()

			tt.runClients(t, 1)

			var telemetry *usmhttp2.HTTP2Telemetry
			assert.Eventually(t, func() bool {
				telemetry, err = usmhttp2.Spec.Instance.(*usmhttp2.Protocol).GetHTTP2KernelTelemetry()
				require.NoError(t, err)
				return reflect.DeepEqual(tt.expectedTelemetry, telemetry)
			}, time.Second*5, time.Millisecond*100)
			if t.Failed() {
				t.Logf("expected telemetry: %+v;\ngot: %+v", tt.expectedTelemetry, telemetry)
			}
		})
	}
}

func getClientsArray(t *testing.T, size int) []*nethttp.Client {
	t.Helper()

	res := make([]*nethttp.Client, size)
	for i := 0; i < size; i++ {
		res[i] = newH2CClient(t)
	}

	return res
}

func startH2CServer(t *testing.T) {
	t.Helper()

	srv := &nethttp.Server{
		Addr: http2SrvPortStr,
		Handler: h2c.NewHandler(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
			w.WriteHeader(200)
			w.Write([]byte("test"))
		}), &http2.Server{}),
		IdleTimeout: 2 * time.Second,
	}

	err := http2.ConfigureServer(srv, nil)
	require.NoError(t, err)

	l, err := net.Listen("tcp", http2SrvPortStr)
	require.NoError(t, err, "could not create listening socket")

	go func() {
		srv.Serve(l)
		require.NoErrorf(t, err, "could not start HTTP2 server")
	}()

	t.Cleanup(func() { srv.Close() })
}

func newH2CClient(t *testing.T) *nethttp.Client {
	t.Helper()

	client := &nethttp.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
		},
	}

	return client
}

func getClientsIndex(index, totalCount int) int {
	return index % totalCount
}

func assertAllRequestsExists(t *testing.T, monitor *Monitor, requests []*nethttp.Request) {
	requestsExist := make([]bool, len(requests))

	assert.Eventually(t, func() bool {
		stats := getHttpStats(t, monitor)

		if len(stats) == 0 {
			return false
		}

		for reqIndex, req := range requests {
			if !requestsExist[reqIndex] {
				exists, err := isRequestIncludedOnce(stats, req)
				require.NoError(t, err)
				requestsExist[reqIndex] = exists
			}
		}

		// Slight optimization here, if one is missing, then go into another cycle of checking the new connections.
		// otherwise, if all present, abort.
		for _, exists := range requestsExist {
			if !exists {
				return false
			}
		}

		return true
	}, 3*time.Second, time.Millisecond*100, "connection not found")

	if t.Failed() {
		o, err := monitor.DumpMaps("http_in_flight")
		if err != nil {
			t.Logf("failed dumping http_in_flight: %s", err)
		} else {
			t.Log(o)
		}

		for reqIndex, exists := range requestsExist {
			if !exists {
				// reqIndex is 0 based, while the number is requests[reqIndex] is 1 based.
				t.Logf("request %d was not found (req %v)", reqIndex+1, requests[reqIndex])
			}
		}
	}
}

func testHTTPMonitor(t *testing.T, targetAddr, serverAddr string, numReqs int, o testutil.Options) {
	monitor := newHTTPMonitor(t)

	srvDoneFn := testutil.HTTPServer(t, serverAddr, o)
	t.Cleanup(srvDoneFn)

	// Perform a number of random requests
	requestFn := requestGenerator(t, targetAddr, emptyBody)
	var requests []*nethttp.Request
	for i := 0; i < numReqs; i++ {
		requests = append(requests, requestFn())
	}
	srvDoneFn()

	// Ensure all captured transactions get sent to user-space
	assertAllRequestsExists(t, monitor, requests)
}

var (
	httpMethods         = []string{nethttp.MethodGet, nethttp.MethodHead, nethttp.MethodPost, nethttp.MethodPut, nethttp.MethodPatch, nethttp.MethodDelete, nethttp.MethodOptions}
	httpMethodsWithBody = []string{nethttp.MethodPost, nethttp.MethodPut, nethttp.MethodPatch, nethttp.MethodDelete}
	statusCodes         = []int{nethttp.StatusOK, nethttp.StatusMultipleChoices, nethttp.StatusBadRequest, nethttp.StatusInternalServerError}
)

func requestGenerator(t *testing.T, targetAddr string, reqBody []byte) func() *nethttp.Request {
	var (
		random  = rand.New(rand.NewSource(time.Now().Unix()))
		idx     = 0
		client  = new(nethttp.Client)
		reqBuf  = make([]byte, 0, len(reqBody))
		respBuf = make([]byte, 512)
	)

	// Disabling http2
	tr := nethttp.DefaultTransport.(*nethttp.Transport).Clone()
	tr.ForceAttemptHTTP2 = false
	tr.TLSNextProto = make(map[string]func(authority string, c *tls.Conn) nethttp.RoundTripper)

	client.Transport = tr

	return func() *nethttp.Request {
		idx++
		var method string
		var body io.Reader
		var finalBody []byte
		if len(reqBody) > 0 {
			finalBody = reqBuf[:0]
			finalBody = append(finalBody, []byte(strings.Repeat(" ", idx))...)
			finalBody = append(finalBody, reqBody...)
			body = bytes.NewReader(finalBody)

			// save resized-buffer
			reqBuf = finalBody

			method = httpMethodsWithBody[random.Intn(len(httpMethodsWithBody))]
		} else {
			method = httpMethods[random.Intn(len(httpMethods))]
		}
		status := statusCodes[random.Intn(len(statusCodes))]
		url := fmt.Sprintf("http://%s/%d/request-%d", targetAddr, status, idx)
		req, err := nethttp.NewRequest(method, url, body)
		require.NoError(t, err)

		resp, err := client.Do(req)
		if strings.Contains(targetAddr, "ignore") {
			return req
		}
		require.NoError(t, err)
		defer resp.Body.Close()
		if len(reqBody) > 0 {
			for {
				n, err := resp.Body.Read(respBuf)
				require.True(t, n <= len(finalBody))
				require.Equal(t, respBuf[:n], finalBody[:n])
				if err != nil {
					assert.Equal(t, io.EOF, err)
					break
				}
				finalBody = finalBody[n:]
			}
		}
		return req
	}
}

func includesRequest(t *testing.T, allStats map[http.Key]*http.RequestStats, req *nethttp.Request) {
	expectedStatus := testutil.StatusFromPath(req.URL.Path)
	included, err := isRequestIncludedOnce(allStats, req)
	require.NoError(t, err)
	if !included {
		t.Errorf(
			"could not find HTTP transaction matching the following criteria:\n path=%s method=%s status=%d",
			req.URL.Path,
			req.Method,
			expectedStatus,
		)
	}
}

func requestNotIncluded(t *testing.T, allStats map[http.Key]*http.RequestStats, req *nethttp.Request) {
	included, err := isRequestIncludedOnce(allStats, req)
	require.NoError(t, err)
	if included {
		expectedStatus := testutil.StatusFromPath(req.URL.Path)
		t.Errorf(
			"should not find HTTP transaction matching the following criteria:\n path=%s method=%s status=%d",
			req.URL.Path,
			req.Method,
			expectedStatus,
		)
	}
}

func isRequestIncludedOnce(allStats map[http.Key]*http.RequestStats, req *nethttp.Request) (bool, error) {
	occurrences := countRequestOccurrences(allStats, req)

	if occurrences == 1 {
		return true, nil
	} else if occurrences == 0 {
		return false, nil
	}
	return false, fmt.Errorf("expected to find 1 occurrence of %v, but found %d instead", req, occurrences)
}

//nolint:revive // TODO(USM) Fix revive linter
func getHttpStats(t *testing.T, mon *Monitor) map[http.Key]*http.RequestStats {
	t.Helper()

	allStats := mon.GetProtocolStats()
	require.NotNil(t, allStats)

	httpStats, ok := allStats[protocols.HTTP]
	require.True(t, ok)

	return httpStats.(map[http.Key]*http.RequestStats)
}

func countRequestOccurrences(allStats map[http.Key]*http.RequestStats, req *nethttp.Request) int {
	expectedStatus := testutil.StatusFromPath(req.URL.Path)
	occurrences := 0
	for key, stats := range allStats {
		if key.Path.Content.Get() != req.URL.Path {
			continue
		}
		if requests, exists := stats.Data[expectedStatus]; exists && requests.Count > 0 {
			occurrences++
		}
	}

	return occurrences
}

func newHTTPMonitorWithCfg(t *testing.T, cfg *networkconfig.Config) *Monitor {
	cfg.EnableHTTPMonitoring = true

	monitor, err := NewMonitor(cfg, nil, nil, nil)
	skipIfNotSupported(t, err)
	require.NoError(t, err)
	t.Cleanup(func() {
		monitor.Stop()
		libtelemetry.Clear()
	})

	// at this stage the test can be legitimately skipped due to missing BTF information
	// in the context of CO-RE
	err = monitor.Start()
	skipIfNotSupported(t, err)
	require.NoError(t, err)
	return monitor
}

func newHTTPMonitor(t *testing.T) *Monitor {
	return newHTTPMonitorWithCfg(t, networkconfig.New())
}

func skipIfNotSupported(t *testing.T, err error) {
	notSupported := new(errNotSupported)
	if errors.As(err, &notSupported) {
		t.Skipf("skipping test because this kernel is not supported: %s", notSupported)
	}
}
