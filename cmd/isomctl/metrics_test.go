package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/PapyrusVIP/isomer/internal/testutil"
)

func TestMetrics(t *testing.T) {
	netns := mustReadyNetNS(t)

	isomctl := isomctlTestCall{
		NetNS:     netns,
		Cmd:       "metrics",
		Args:      []string{"127.0.0.1", "0"},
		Listeners: make(chan net.Listener, 1),
	}

	isomctl.Start(t)

	var ln net.Listener
	select {
	case ln = <-isomctl.Listeners:
	case <-time.After(time.Second):
		t.Fatal("isomctl isn't listening after one second")
	}

	client := http.Client{Timeout: 5 * time.Second}
	addr := fmt.Sprintf("http://%s/metrics", ln.Addr().String())
	for i := 0; i < 3; i++ {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			res, err := client.Get(addr)
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()

			body, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal("Can't ready body:", err)
			}

			if !bytes.Contains(body, []byte("# HELP ")) {
				t.Error("Output doesn't contain prometheus export format")
			}

			if !bytes.Contains(body, []byte("# TYPE isomer_")) {
				t.Error("Output doesn't contain isomer prefix")
			}

			if !bytes.Contains(body, []byte("# TYPE build_info")) {
				t.Error("Output doesn't contain unprefixed build_info")
			}
		})
	}
}

func TestMetricsInvalidArgs(t *testing.T) {
	netns := testutil.CurrentNetNS(t)

	_, err := testIsomctl(t, netns, "metrics")
	if err == nil {
		t.Error("metrics command accepts no arguments")
	}

	_, err = testIsomctl(t, netns, "metrics", "127.0.0.1")
	if err == nil {
		t.Error("metrics command accepts missing port")
	}
}
