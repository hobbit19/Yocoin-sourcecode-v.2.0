// Authored and revised by YOC team, 2018
// License placeholder #1

package simulation

import (
	"fmt"
	"net/http"

	"github.com/Yocoin15/Yocoin_Sources/log"
	"github.com/Yocoin15/Yocoin_Sources/p2p/simulations"
)

// Package defaults.
var (
	DefaultHTTPSimAddr = ":8888"
)

//WithServer implements the builder pattern constructor for Simulation to
//start with a HTTP server
func (s *Simulation) WithServer(addr string) *Simulation {
	//assign default addr if nothing provided
	if addr == "" {
		addr = DefaultHTTPSimAddr
	}
	log.Info(fmt.Sprintf("Initializing simulation server on %s...", addr))
	//initialize the HTTP server
	s.handler = simulations.NewServer(s.Net)
	s.runC = make(chan struct{})
	//add swarm specific routes to the HTTP server
	s.addSimulationRoutes()
	s.httpSrv = &http.Server{
		Addr:    addr,
		Handler: s.handler,
	}
	go func() {
		err := s.httpSrv.ListenAndServe()
		if err != nil {
			log.Error("Error starting the HTTP server", "error", err)
		}
	}()
	return s
}

//register additional HTTP routes
func (s *Simulation) addSimulationRoutes() {
	s.handler.POST("/runsim", s.RunSimulation)
}

// RunSimulation is the actual POST endpoint runner
func (s *Simulation) RunSimulation(w http.ResponseWriter, req *http.Request) {
	log.Debug("RunSimulation endpoint running")
	s.runC <- struct{}{}
	w.WriteHeader(http.StatusOK)
}
