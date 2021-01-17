// Copyright 2017 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/pprof"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

const defaultStatusAddr = ":10080"

func (s *Server) startStatusHTTP() {
	go s.startHTTPServer()
}

func (s *Server) startHTTPServer() {
	router := mux.NewRouter()

	addr := fmt.Sprintf(":%d", s.cfg.Status.StatusPort)
	if s.cfg.Status.StatusPort == 0 {
		addr = defaultStatusAddr
	}

	serverMux := http.NewServeMux()
	serverMux.Handle("/", router)

	serverMux.HandleFunc("/debug/pprof/", pprof.Index)
	serverMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	serverMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	serverMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	serverMux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	var (
		err            error
		httpRouterPage bytes.Buffer
		pathTemplate   string
	)
	httpRouterPage.WriteString("<html><head><title>TiDB Status and Metrics Report</title></head><body><h1>TiDB Status and Metrics Report</h1><table>")
	err = router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		pathTemplate, err = route.GetPathTemplate()
		if err != nil {
			log.Error("Get http router path error ", err)
		}
		name := route.GetName() //If the name attribute is not set, GetName returns ""
		if name != "" && err == nil {
			httpRouterPage.WriteString("<tr><td><a href='" + pathTemplate + "'>" + name + "</a><td></tr>")
		}
		return nil
	})
	if err != nil {
		log.Error("Generate root error ", err)
	}
	httpRouterPage.WriteString("<tr><td><a href='/debug/pprof/'>Debug</a><td></tr>")
	httpRouterPage.WriteString("</table></body></html>")
	router.HandleFunc("/", func(responseWriter http.ResponseWriter, request *http.Request) {
		_, err = responseWriter.Write(httpRouterPage.Bytes())
		if err != nil {
			log.Error("Http index page error ", err)
		}
	})

	log.Infof("Listening on %v for status and metrics report.", addr)
	s.statusServer = &http.Server{Addr: addr, Handler: serverMux}

	if len(s.cfg.Security.ClusterSSLCA) != 0 {
		err = s.statusServer.ListenAndServeTLS(s.cfg.Security.ClusterSSLCert, s.cfg.Security.ClusterSSLKey)
	} else {
		err = s.statusServer.ListenAndServe()
	}

	if err != nil {
		log.Info(err)
	}
}
