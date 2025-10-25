package main

import (
	"strings"
	"testing"

	"github.com/PapyrusVIP/isomer/internal"
)

func TestStatus(t *testing.T) {
	netns := mustReadyNetNS(t)

	dp := mustOpenDispatcher(t, netns)
	mustAddBinding(t, dp, "foo", internal.TCP, "::1", 80)
	sock := makeListeningSocket(t, netns, "tcp")
	mustRegisterSocket(t, dp, "foo", sock)
	dp.Close()

	output, err := testIsomctl(t, netns, "status")
	if err != nil {
		t.Fatal("Can't execute status:", err)
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "foo") {
		t.Error("Output of status doesn't contain label foo")
	}

	cookie := mustSocketCookie(t, sock)
	if !strings.Contains(outputStr, cookie.String()) {
		t.Error("Output of status doesn't contain", cookie)
	}

	output2, err := testIsomctl(t, netns, "status")
	if err != nil {
		t.Fatal(err)
	}

	output2Str := output2.String()
	if output2Str != outputStr {
		t.Log(outputStr)
		t.Log(output2Str)
		t.Error("The output of list isn't stable across invocations")
	}
}

func TestStatusFilteredByLabel(t *testing.T) {
	netns := mustReadyNetNS(t)

	dp := mustOpenDispatcher(t, netns)
	mustAddBinding(t, dp, "foo", internal.TCP, "::1", 80)
	sock := makeListeningSocket(t, netns, "tcp")
	mustRegisterSocket(t, dp, "foo", sock)
	dp.Close()

	output, err := testIsomctl(t, netns, "status", "foo")
	if err != nil {
		t.Fatal("Can't execute list foo:", err)
	}

	if !strings.Contains(output.String(), "foo") {
		t.Error("Output of list doesn't contain label foo")
	}

	output, err = testIsomctl(t, netns, "status", "bar")
	if err != nil {
		t.Fatal("Can't execute list bar:", err)
	}

	if strings.Contains(output.String(), "foo") {
		t.Error("Output of list contains label foo, even though it should be filtered")
	}
}
