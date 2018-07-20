package cmd

import (
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/coreos/go-systemd/activation"
	"github.com/dcos/dcos-check-runner/api"
	"github.com/dcos/dcos-check-runner/runner"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var httpServerCmd = &cobra.Command{
	Use:   "http-server",
	Short: "Start the check runner HTTP server",
	Run: func(cmd *cobra.Command, args []string) {
		r, err := runner.NewRunner(defaultConfig.FlagRole)
		if err != nil {
			logrus.Fatal(err)
		}

		if err := r.LoadFromFile(checkCfgFile); err != nil {
			logrus.Fatal(err)
		}

		// Set up environment for running check commands.
		for k, v := range r.CheckEnv {
			os.Setenv(k, v)
		}

		router := api.NewRouter(r, defaultConfig.FlagBaseURI)
		var serveErr error
		if defaultConfig.FlagSystemdSocket {
			listener, err := getSystemdSocket()
			if err != nil {
				logrus.Fatal(err)
			}
			logrus.Infof("Listening at %s", listener.Addr().String())
			serveErr = http.Serve(listener, router)
		} else {
			addr := fmt.Sprintf("%s:%d", defaultConfig.FlagHost, defaultConfig.FlagPort)
			logrus.Infof("Listening at %s", addr)
			serveErr = http.ListenAndServe(addr, router)
		}

		logrus.Fatal(serveErr)
	},
}

func init() {
	RootCmd.AddCommand(httpServerCmd)
	httpServerCmd.PersistentFlags().StringVarP(&defaultConfig.FlagHost, "host", "a", "0.0.0.0", "Server's host")
	httpServerCmd.PersistentFlags().IntVarP(&defaultConfig.FlagPort, "port", "p", 8000, "Server's TCP port")
	httpServerCmd.PersistentFlags().BoolVar(&defaultConfig.FlagSystemdSocket, "systemd-socket", false, "Listen on systemd socket")
	httpServerCmd.PersistentFlags().StringVar(&defaultConfig.FlagBaseURI, "base-uri", "", "Server's base URI")
}

func getSystemdSocket() (net.Listener, error) {
	listeners, err := activation.Listeners()
	if err != nil {
		return nil, errors.Wrap(err, "Error getting systemd socket")
	}
	if len(listeners) != 1 {
		return nil, errors.New(fmt.Sprintf("Expected 1 systemd socket, found %d", len(listeners)))
	}
	if listeners[0] == nil {
		return nil, errors.New("No systemd socket found")
	}
	return listeners[0], nil
}
