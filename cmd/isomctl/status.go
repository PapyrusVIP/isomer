package main

import (
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/PapyrusVIP/isomer/internal"
)

func status(e *env, args ...string) error {
	set := e.newFlagSet("status", "--", "label")
	set.Description = "Show current bindings and destinations."
	if err := set.Parse(args); err != nil {
		return err
	}

	var (
		bindings internal.Bindings
		dests    []internal.Destination
		cookies  map[internal.Destination]internal.SocketCookie
		metrics  *internal.Metrics
	)
	{
		dp, err := e.openDispatcher(true)
		if err != nil {
			return err
		}
		defer dp.Close()

		bindings, err = dp.Bindings()
		if err != nil {
			return fmt.Errorf("can't get bindings: %s", err)
		}

		dests, cookies, err = dp.Destinations()
		if err != nil {
			return fmt.Errorf("get destinations: %s", err)
		}

		metrics, err = dp.Metrics()
		if err != nil {
			return fmt.Errorf("get metrics: %s", err)
		}

		dp.Close()
	}

	if label := set.Arg(0); label != "" {
		var filtered internal.Bindings
		for _, bind := range bindings {
			if bind.Label == label {
				filtered = append(filtered, bind)
			}
		}
		bindings = filtered

		var filteredDests []internal.Destination
		for _, dest := range dests {
			if dest.Label == label {
				filteredDests = append(filteredDests, dest)
			}
		}
		dests = filteredDests
	}

	w := tabwriter.NewWriter(e.stdout, 0, 0, 1, ' ', tabwriter.AlignRight)

	e.stdout.Log("Bindings:")
	if err := printBindings(w, bindings); err != nil {
		return err
	}

	sortDestinations(dests)

	e.stdout.Log("\nDestinations:")
	fmt.Fprintln(w, "label\tdomain\tprotocol\tsocket\tlookups\tmisses\terrors\t")

	for _, dest := range dests {
		destMetrics := metrics.Destinations[dest]
		_, err := fmt.Fprint(w,
			dest.Label, "\t",
			dest.Domain, "\t",
			dest.Protocol, "\t",
			cookies[dest], "\t",
			destMetrics.Lookups, "\t",
			destMetrics.Misses, "\t",
			destMetrics.TotalErrors(), "\t",
			"\n",
		)
		if err != nil {
			return err
		}
	}

	if err := w.Flush(); err != nil {
		return err
	}

	return nil
}

func printBindings(w *tabwriter.Writer, bindings internal.Bindings) error {
	// Output from most specific to least specific.
	sort.Sort(bindings)

	fmt.Fprintln(w, "protocol\tprefix\tport\tlabel\t")

	for _, bind := range bindings {
		_, err := fmt.Fprintf(w, "%v\t%s\t%d\t%s\t\n", bind.Protocol, bind.Prefix, bind.Port, bind.Label)
		if err != nil {
			return err
		}
	}

	return w.Flush()
}

func sortDestinations(dests []internal.Destination) {
	sort.Slice(dests, func(i, j int) bool {
		a, b := dests[i], dests[j]
		if a.Label != b.Label {
			return a.Label < b.Label
		}

		if a.Domain != b.Domain {
			return a.Domain < b.Domain
		}

		return a.Protocol < b.Protocol
	})
}
