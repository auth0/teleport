/*
Copyright 2016 Gravitational, Inc.

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
package reversetunnel

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/gravitational/teleport"
	"github.com/gravitational/teleport/lib/auth"
	"github.com/gravitational/teleport/lib/services"

	log "github.com/Sirupsen/logrus"
	"github.com/gravitational/trace"
)

func newlocalSite(domainName string, client auth.ClientI) *localSite {
	return &localSite{
		client:     client,
		domainName: domainName,
		log: log.WithFields(log.Fields{
			teleport.Component: teleport.ComponentReverseTunnel,
			teleport.ComponentFields: map[string]string{
				"domainName": domainName,
				"side":       "server",
				"type":       "localSite",
			},
		}),
	}
}

// localSite allows to directly access the remote servers
// not using any tunnel, and using standard SSH
//
// it implements RemoteSite interface
type localSite struct {
	sync.Mutex
	client auth.ClientI

	authServer  string
	log         *log.Entry
	domainName  string
	connections []*remoteConn
	lastUsed    int
	lastActive  time.Time
	srv         *server
}

func (s *localSite) GetClient() (auth.ClientI, error) {
	return s.client, nil
}

func (s *localSite) String() string {
	return fmt.Sprintf("localSite(%v)", s.domainName)
}

func (s *localSite) GetStatus() string {
	return RemoteSiteStatusOnline
}

func (s *localSite) GetName() string {
	return s.domainName
}

func (s *localSite) GetLastConnected() time.Time {
	return time.Now()
}

// Dial dials a given host in this site (cluster).
func (s *localSite) Dial(from net.Addr, to net.Addr) (net.Conn, error) {
	s.log.Debugf("[PROXY] localSite.Dial(from=%v, to=%v)", from, to)
	return net.Dial(to.Network(), to.String())
}

func findServer(addr string, servers []services.Server) (services.Server, error) {
	for i := range servers {
		srv := servers[i]
		_, port, err := net.SplitHostPort(srv.GetAddr())
		if err != nil {
			log.Warningf("server %v(%v) has incorrect address format (%v)",
				srv.GetAddr(), srv.GetHostname(), err.Error())
		} else {
			if (len(srv.GetHostname()) != 0) && (len(port) != 0) && (addr == srv.GetHostname()+":"+port || addr == srv.GetAddr()) {
				return srv, nil
			}
		}
	}
	return nil, trace.NotFound("server %v is unknown", addr)
}
