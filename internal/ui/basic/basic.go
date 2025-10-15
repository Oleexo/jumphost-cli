package basic

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/Oleexo/jumphost-cli/internal/awsclient"
	"github.com/Oleexo/jumphost-cli/internal/ui/connectflow"
)

// RunConnectFlowBasic provides a line-oriented fallback for environments without a TTY.
// It mimics the connectflow.Result contract using plain stdin/stdout prompts.
func RunConnectFlowBasic(ctx context.Context, rds *awsclient.RDS, in io.Reader, out io.Writer) (
	connectflow.Result,
	error) {
	w := bufio.NewWriter(out)
	defer func() { _ = w.Flush() }()
	bw := bufio.NewReader(in)

	_, _ = fmt.Fprintln(w, "(basic mode) Loading RDS endpoints...")
	endpoints, err := rds.ListEndpoints(ctx)
	if err != nil {
		return connectflow.Result{}, err
	}
	if len(endpoints) == 0 {
		return connectflow.Result{}, fmt.Errorf("no RDS endpoints found")
	}
	for i, ep := range endpoints {
		// Build descriptive label similar to TUI mode
		label := fmt.Sprintf("%s:%d", ep.Address, ep.Port)
		desc := ""
		if ep.DBInstanceID != "" {
			desc = ep.DBInstanceID
		}
		if ep.Engine != "" {
			if desc != "" {
				desc += " | "
			}
			desc += ep.Engine
			if ep.EngineVersion != "" {
				desc += " " + ep.EngineVersion
			}
		}
		if ep.DBInstanceStatus != "" {
			if desc != "" {
				desc += " | "
			}
			desc += ep.DBInstanceStatus
		}
		if ep.DBInstanceClass != "" {
			if desc != "" {
				desc += " | "
			}
			desc += ep.DBInstanceClass
		}
		if desc != "" {
			_, _ = fmt.Fprintf(w, " %d) %s - %s\n", i+1, label, desc)
		} else {
			_, _ = fmt.Fprintf(w, " %d) %s\n", i+1, label)
		}
	}
	_, _ = fmt.Fprintf(w, "Select endpoint [1-%d] (blank=1, q=quit): ", len(endpoints))
	_ = w.Flush()
	choice, _ := bw.ReadString('\n')
	choice = strings.TrimSpace(choice)
	if choice == "q" || choice == "Q" {
		return connectflow.Result{Cancelled: true}, nil
	}
	idx := 1
	if choice != "" {
		v, convErr := strconv.Atoi(choice)
		if convErr != nil || v < 1 || v > len(endpoints) {
			return connectflow.Result{}, fmt.Errorf("invalid selection")
		}
		idx = v
	}
	sel := endpoints[idx-1]
	res := connectflow.Result{Endpoint: sel.Address, RemotePort: sel.Port}

	_, _ = fmt.Fprintf(w, "Local port (blank=%d): ", sel.Port)
	_ = w.Flush()
	lpStr, _ := bw.ReadString('\n')
	lpStr = strings.TrimSpace(lpStr)
	if lpStr == "" {
		res.LocalPort = sel.Port
	} else {
		p, convErr := strconv.Atoi(lpStr)
		if convErr != nil || p <= 0 || p > 65535 {
			return connectflow.Result{}, fmt.Errorf("invalid local port")
		}
		res.LocalPort = p
	}

	_, _ = fmt.Fprintf(w, "Add /etc/hosts entry mapping %s to 127.0.0.1? (Y/n): ", res.Endpoint)
	_ = w.Flush()
	ans, _ := bw.ReadString('\n')
	ans = strings.TrimSpace(strings.ToLower(ans))
	res.ApplyHosts = ans != "n" && ans != "no"

	return res, nil
}

// RunJumpHostSelectBasic prompts user to select a jumphost instance ID.
// Returns selected ID, cancelled flag, error.
func RunJumpHostSelectBasic(_ context.Context, ids []string, in io.Reader, out io.Writer) (string, bool, error) {
	w := bufio.NewWriter(out)
	defer func() { _ = w.Flush() }()
	bw := bufio.NewReader(in)

	if len(ids) == 0 {
		return "", false, fmt.Errorf("no jumphost instances to select")
	}
	_, _ = fmt.Fprintln(w, "(basic mode) Select jumphost instance:")
	for i, id := range ids {
		_, _ = fmt.Fprintf(w, " %d) %s\n", i+1, id)
	}
	_, _ = fmt.Fprintf(w, "Choice [1-%d] (blank=1, q=quit): ", len(ids))
	_ = w.Flush()
	choice, _ := bw.ReadString('\n')
	choice = strings.TrimSpace(choice)
	if choice == "q" || choice == "Q" {
		return "", true, nil
	}
	idx := 1
	if choice != "" {
		v, err := strconv.Atoi(choice)
		if err != nil || v < 1 || v > len(ids) {
			return "", false, fmt.Errorf("invalid selection")
		}
		idx = v
	}
	return ids[idx-1], false, nil
}
