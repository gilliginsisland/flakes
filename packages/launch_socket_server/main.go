package main

import (
	"log"
	"net"
	"net/netip"
	"os"
	"os/exec"
	"time"

	"launch_socket_server/launch"
	"launch_socket_server/proxy"
)

// getenv looks up the environment var, returning a fallback if not set
func getenv(key string, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("usage: %s <program> [<arg> ...]", os.Args[0])
	}

	address := getenv("LAUNCH_CMD_ADDRESS", "127.0.0.1:0")
	ipp, err := netip.ParseAddrPort(address)
	if err != nil {
		log.Fatalf("invalid LAUNCH_CMD_ADDRESS: %s", err.Error())
	}

	socket := getenv("LAUNCH_DAEMON_SOCKET_NAME", "Socket")
	listeners, err := launch.ActivateSocket(socket)
	if err != nil || len(listeners) == 0 {
		log.Fatalf("error activating launch socket: %s", err)
	}

	dst := net.TCPAddrFromAddrPort(ipp)
	if dst.Port == 0 {
		if dst.Port, err = proxy.GetFreePort(dst.IP); err != nil {
			log.Fatalf("error getting free port: %s", err.Error())
		}
		os.Setenv("LAUNCH_CMD_ADDRESS", dst.String())
	}

	cmd := exec.Command(os.Args[1], os.Args[2:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		log.Fatalf("error starting command: %s", err)
	}

	go func() {
		if err := cmd.Wait(); err != nil {
			log.Fatalf("command exited with error: %s", err)
		}
		log.Println("command exited")
		os.Exit(0)
	}()

	p := proxy.New(dst)
	for !p.Reachable() {
		time.Sleep(2 * time.Second)
	}

	for _, listener := range listeners {
		go p.Serve(listener)
	}

	select {}
}
