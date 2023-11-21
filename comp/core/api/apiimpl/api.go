// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023-present Datadog, Inc.

/*
Package apiimpl implements the agent IPC api. Using HTTP
calls, it's possible to communicate with the agent,
sending commands and receiving infos.
*/
package apiimpl

import (
	"context"
	"crypto/tls"
	"fmt"
	stdLog "log"
	"net"
	"net/http"
	"time"

	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/DataDog/datadog-agent/comp/core/api"
	"github.com/DataDog/datadog-agent/comp/core/api/apiimpl/internal/agent"
	"github.com/DataDog/datadog-agent/comp/core/api/apiimpl/internal/check"
	"github.com/DataDog/datadog-agent/comp/core/flare"
	"github.com/DataDog/datadog-agent/comp/dogstatsd/replay"
	dogstatsdServer "github.com/DataDog/datadog-agent/comp/dogstatsd/server"
	dogstatsddebug "github.com/DataDog/datadog-agent/comp/dogstatsd/serverDebug"
	logsAgent "github.com/DataDog/datadog-agent/comp/logs/agent"
	"github.com/DataDog/datadog-agent/comp/metadata/host"
	"github.com/DataDog/datadog-agent/comp/metadata/inventoryagent"
	"github.com/DataDog/datadog-agent/pkg/aggregator/sender"
	"github.com/DataDog/datadog-agent/pkg/api/util"
	"github.com/DataDog/datadog-agent/pkg/config"
	remoteconfig "github.com/DataDog/datadog-agent/pkg/config/remote/service"
	pb "github.com/DataDog/datadog-agent/pkg/proto/pbgo/core"
	"github.com/DataDog/datadog-agent/pkg/tagger"
	taggerserver "github.com/DataDog/datadog-agent/pkg/tagger/server"
	pkgUtil "github.com/DataDog/datadog-agent/pkg/util"
	"github.com/DataDog/datadog-agent/pkg/util/fxutil"
	grpcutil "github.com/DataDog/datadog-agent/pkg/util/grpc"
	"github.com/DataDog/datadog-agent/pkg/workloadmeta"
	workloadmetaServer "github.com/DataDog/datadog-agent/pkg/workloadmeta/server"
	"github.com/cihub/seelog"
	gorilla "github.com/gorilla/mux"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
)

// Module defines the fx options for this component.
var Module = fxutil.Component(
	fx.Provide(newAPIServer),
)

type apiServer struct {
	listener net.Listener
}

var _ api.Component = (*apiServer)(nil)

func newAPIServer() api.Component {
	return &apiServer{}
}

// StartServer creates the router and starts the HTTP server
func (agentAPI *apiServer) StartServer(
	configService *remoteconfig.Service,
	flare flare.Component,
	dogstatsdServer dogstatsdServer.Component,
	capture replay.Component,
	serverDebug dogstatsddebug.Component,
	logsAgent pkgUtil.Optional[logsAgent.Component],
	senderManager sender.DiagnoseSenderManager,
	hostMetadata host.Component,
	invAgent inventoryagent.Component,
) error {
	initializeTLS()

	// get the transport we're going to use under HTTP
	var err error
	agentAPI.listener, err = getListener()
	if err != nil {
		// we use the listener to handle commands for the Agent, there's
		// no way we can recover from this error
		return fmt.Errorf("Unable to create the api server: %v", err)
	}

	err = util.CreateAndSetAuthToken()
	if err != nil {
		return err
	}

	// gRPC server
	authInterceptor := grpcutil.AuthInterceptor(parseToken)
	opts := []grpc.ServerOption{
		grpc.Creds(credentials.NewClientTLSFromCert(tlsCertPool, tlsAddr)),
		grpc.StreamInterceptor(grpc_auth.StreamServerInterceptor(authInterceptor)),
		grpc.UnaryInterceptor(grpc_auth.UnaryServerInterceptor(authInterceptor)),
	}

	s := grpc.NewServer(opts...)
	pb.RegisterAgentServer(s, &server{})
	pb.RegisterAgentSecureServer(s, &serverSecure{
		configService:      configService,
		taggerServer:       taggerserver.NewServer(tagger.GetDefaultTagger()),
		workloadmetaServer: workloadmetaServer.NewServer(workloadmeta.GetGlobalStore()),
		dogstatsdServer:    dogstatsdServer,
		capture:            capture,
	})

	dcreds := credentials.NewTLS(&tls.Config{
		ServerName: tlsAddr,
		RootCAs:    tlsCertPool,
	})
	dopts := []grpc.DialOption{grpc.WithTransportCredentials(dcreds)}

	// starting grpc gateway
	ctx := context.Background()
	gwmux := runtime.NewServeMux()
	err = pb.RegisterAgentHandlerFromEndpoint(
		ctx, gwmux, tlsAddr, dopts)
	if err != nil {
		panic(err)
	}

	err = pb.RegisterAgentSecureHandlerFromEndpoint(
		ctx, gwmux, tlsAddr, dopts)
	if err != nil {
		panic(err)
	}

	// Setup multiplexer
	// create the REST HTTP router
	agentMux := gorilla.NewRouter()
	checkMux := gorilla.NewRouter()
	// Validate token for every request
	agentMux.Use(validateToken)
	checkMux.Use(validateToken)

	mux := http.NewServeMux()
	mux.Handle(
		"/agent/",
		http.StripPrefix("/agent",
			agent.SetupHandlers(
				agentMux,
				flare,
				dogstatsdServer,
				serverDebug,
				logsAgent,
				senderManager,
				hostMetadata,
				invAgent,
			)))
	mux.Handle("/check/", http.StripPrefix("/check", check.SetupHandlers(checkMux)))
	mux.Handle("/", gwmux)

	// Use a stack depth of 4 on top of the default one to get a relevant filename in the stdlib
	logWriter, _ := config.NewLogWriter(5, seelog.ErrorLvl)

	srv := grpcutil.NewMuxedGRPCServer(
		tlsAddr,
		&tls.Config{
			Certificates: []tls.Certificate{*tlsKeyPair},
			NextProtos:   []string{"h2"},
			MinVersion:   tls.VersionTLS12,
		},
		s,
		grpcutil.TimeoutHandlerFunc(mux, time.Duration(config.Datadog.GetInt64("server_timeout"))*time.Second),
	)

	srv.ErrorLog = stdLog.New(logWriter, "Error from the agent http API server: ", 0) // log errors to seelog

	tlsListener := tls.NewListener(agentAPI.listener, srv.TLSConfig)

	go srv.Serve(tlsListener) //nolint:errcheck
	return nil
}

// StopServer closes the connection and the server
// stops listening to new commands.
func (agentAPI *apiServer) StopServer() {
	if agentAPI.listener != nil {
		agentAPI.listener.Close()
	}
}

// ServerAddress retruns the server address.
func (agentAPI *apiServer) ServerAddress() *net.TCPAddr {
	return agentAPI.listener.Addr().(*net.TCPAddr)
}
