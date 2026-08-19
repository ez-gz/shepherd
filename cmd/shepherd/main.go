package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/ez-gz/shepherd/internal/config"
	"github.com/ez-gz/shepherd/internal/control"
	"github.com/ez-gz/shepherd/internal/env"
	"github.com/ez-gz/shepherd/internal/format"
	"github.com/ez-gz/shepherd/internal/runner"
	"github.com/ez-gz/shepherd/internal/shepherd"
	"github.com/ez-gz/shepherd/internal/supervisor"
	"github.com/ez-gz/shepherd/internal/transcript"
	"github.com/ez-gz/shepherd/internal/ui"
	"github.com/ez-gz/shepherd/internal/workstream"
)

var version = "0.8.0"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "__statusline" {
		// Instrumentation is observational and must never break the runner that
		// invoked it. Malformed input or a disappearing pane therefore produces
		// no output and exits successfully.
		if len(os.Args) == 3 && os.Args[2] == "claude" {
			if update, err := runner.ReadClaudeStatus(os.Stdin); err == nil {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				_ = supervisor.PublishNativeStatus(ctx, update)
				cancel()
				if update.Display != "" {
					fmt.Fprintln(os.Stdout, update.Display)
				}
			}
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "__agent" {
		if len(os.Args) != 8 {
			fmt.Fprintln(os.Stderr, "shepherd: invalid internal runner invocation")
			os.Exit(2)
		}
		if err := runner.ExecEncoded(os.Args[2], os.Args[3], os.Args[4], os.Args[5], os.Args[6], os.Args[7]); err != nil {
			fmt.Fprintln(os.Stderr, "shepherd:", err)
			os.Exit(127)
		}
		return
	}

	ensurePilotDocs(os.Stderr)

	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "shepherd:", format.OneLine(err.Error()))
		os.Exit(1)
	}
}

// app is everything a command handler needs from the process around it: two
// places to write, and a way to reach the controller, the settings, and the
// working directory.
//
// Handlers used to take these from package scope — os.Stdout directly, and a
// controller each one built for itself out of the environment. That made every
// verb untestable except by building the binary and running it, which is why
// the end-to-end suite exists and why cmd/shepherd reported almost no coverage. The
// suite still earns its keep, because it tests what actually ships. But
// argument handling, refusal wording and the shape of --json do not need a tmux
// server to check, and with this struct they no longer ask for one.
type app struct {
	out io.Writer
	err io.Writer

	// dial is deliberately called inside each handler rather than once up
	// front. A verb that refuses its arguments must fail on the arguments, not
	// on a missing tmux server, so nothing may dial before the arguments are
	// known to be good.
	dial     func(socket string) (control.Service, error)
	settings func() (config.Store, config.Config, error)
	workdir  func() string

	// transcripts reads what a runner recorded. Its zero value points at the
	// real location under the user's home, so only a test ever sets it.
	transcripts transcript.Reader
}

// newApp wires the real process. Every field here reads the environment; every
// field is replaceable in a test.
func newApp() *app {
	return &app{
		out:      os.Stdout,
		err:      os.Stderr,
		settings: loadSettings,
		workdir:  mustWorkingDirectory,
		dial: func(socket string) (control.Service, error) {
			_, controller, _, err := newController(socket)
			return controller, err
		},
	}
}

func run(args []string) error {
	return newApp().run(args)
}

func (a *app) run(args []string) error {
	if routeGlobalCommand(args, a.out) {
		return nil
	}
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return a.runDashboard(args)
	}
	switch args[0] {
	case "doctor":
		return a.runDoctor(args[1:])
	case "quickstart":
		return a.runQuickstart(args[1:])
	case "list", "ls":
		return a.runList(args[1:])
	case "spawn", "new":
		return a.runSpawn(args[1:])
	case "send":
		return a.runSend(args[1:])
	case "attach":
		return a.runAttach(args[1:])
	case "stop", "rm":
		return a.runStop(args[1:])
	case "peek":
		return a.runPeek(args[1:])
	case "history":
		return a.runHistory(args[1:])
	case "resume":
		return a.runResume(args[1:])
	case "fork":
		return a.runFork(args[1:])
	case "conversation":
		return a.runConversation(args[1:])
	case "ws", "workstream":
		return a.runWorkstreamCommand(args[1:])
	case "title":
		return a.runTitle(args[1:])
	case "move":
		return a.runMove(args[1:])
	case "reorder":
		return a.runSessionReorder(args[1:])
	case "adopt":
		return a.runAdopt(args[1:])
	case "delete":
		return a.runDelete(args[1:])
	case "init":
		return a.runInit(args[1:])
	default:
		return fmt.Errorf("unknown command %q; run shepherd help", args[0])
	}
}

func routeGlobalCommand(args []string, writer io.Writer) bool {
	if len(args) == 0 {
		return false
	}
	switch args[0] {
	case "help", "-h", "--help":
		printHelp(writer)
		return true
	case "version", "--version":
		fmt.Fprintln(writer, "shepherd", version)
		return true
	default:
		return false
	}
}

func (a *app) runDashboard(args []string) error {
	return a.runDashboardSelected(args, "")
}

func (a *app) runDashboardSelected(args []string, selectedSessionID string) error {
	configStore, settings, err := a.settings()
	if err != nil {
		return err
	}
	flags := a.newFlagSet("shepherd")
	root := flags.String("root", a.workdir(), "root directory for newly spawned agents")
	flags.StringVar(root, "C", *root, "root directory for newly spawned agents")
	runnerValue := flags.String("runner", string(settings.DefaultRunner), "default runner: codex, claude, or no-agent")
	flags.StringVar(runnerValue, "r", *runnerValue, "default runner: codex, claude, or no-agent")
	socket := flags.String("socket", defaultSocket(), "private tmux socket name")
	flags.Usage = func() { printHelp(flags.Output()) }
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(flags.Args(), " "))
	}
	backend, err := shepherd.ParseBackend(*runnerValue)
	if err != nil {
		return err
	}
	absRoot, err := filepath.Abs(*root)
	if err != nil {
		return fmt.Errorf("resolve root: %w", err)
	}
	rootInfo, err := os.Stat(absRoot)
	if err != nil {
		return fmt.Errorf("open root %q: %w", absRoot, err)
	}
	if !rootInfo.IsDir() {
		return fmt.Errorf("root %q is not a directory", absRoot)
	}
	manager, controller, stateStore, err := newController(*socket)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if err := manager.Bootstrap(ctx); err != nil {
		return err
	}
	provisionInstallation(ctx, controller, stateStore, a.err)

	program := tea.NewProgram(ui.NewWithSelectedSession(controller, absRoot, backend, configStore, settings, selectedSessionID))
	if _, err := program.Run(); err != nil {
		return fmt.Errorf("run dashboard: %w", err)
	}
	return nil
}

func (a *app) runQuickstart(args []string) error {
	flags := a.newFlagSet("shepherd quickstart")
	socket := flags.String("socket", defaultSocket(), "private tmux socket name")
	flags.Usage = func() {
		fmt.Fprintln(flags.Output(), "Usage: shepherd quickstart")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("usage: shepherd quickstart")
	}
	_, settings, err := a.settings()
	if err != nil {
		return err
	}
	projectRoot, err := filepath.Abs(a.workdir())
	if err != nil {
		return fmt.Errorf("resolve dashboard root: %w", err)
	}
	if _, err := runner.ResolveCommand(shepherd.BackendClaude, settings.Command(shepherd.BackendClaude)); err != nil {
		return fmt.Errorf("quickstart requires Claude: %w", err)
	}
	dir, _, err := installPilotDocs(false)
	if err != nil {
		return err
	}
	stateStore, err := workstream.DefaultStore()
	if err != nil {
		return err
	}
	controller, err := a.dial(*socket)
	if err != nil {
		return err
	}
	startContext, cancelStart := context.WithTimeout(context.Background(), 8*time.Second)
	session, err := launchQuickstart(startContext, controller, stateStore, dir, a.err)
	cancelStart()
	if err != nil {
		return err
	}

	fmt.Fprintf(a.out, "started guided Claude session %s in %s\n", format.ShortID(session.ID), dir)
	fmt.Fprintln(a.err, "attaching now · your first lesson is Ctrl-b, release, then d")
	attachContext, cancelAttach := context.WithTimeout(context.Background(), 5*time.Second)
	command, err := controller.AttachCommand(attachContext, session.ID)
	cancelAttach()
	if err != nil {
		return fmt.Errorf("attach guide %s: %w (the session is still available in shepherd)", format.ShortID(session.ID), err)
	}
	if err := command.Run(); err != nil {
		return fmt.Errorf("attach guide %s: %w (the session is still available in shepherd)", format.ShortID(session.ID), err)
	}

	fmt.Fprintln(a.err, "detached · opening shepherd with the guide selected")
	return a.runDashboardSelected([]string{
		"--runner", string(shepherd.BackendClaude),
		"--root", projectRoot,
		"--socket", *socket,
	}, session.ID)
}

func launchQuickstart(ctx context.Context, controller control.Service, store workstream.FileStore, dir string, writer io.Writer) (control.Session, error) {
	// Provision before Start writes the first session record. The durable state
	// file is the one-time installation marker, so reversing these two actions
	// would permanently suppress the managers workstream on a fresh install.
	provisionInstallation(ctx, controller, store, writer)

	workstreamID := ""
	snapshot, err := controller.Snapshot(ctx)
	if err != nil {
		return control.Session{}, fmt.Errorf("find the %s workstream: %w", ManagersWorkstreamName, err)
	}
	for _, item := range snapshot.Workstreams {
		if item.ArchivedAt == nil && strings.EqualFold(item.Name, ManagersWorkstreamName) {
			workstreamID = item.ID
			break
		}
	}

	document := filepath.Join(dir, quickstartDocumentName)
	session, err := controller.Start(ctx, control.StartRequest{
		Backend: shepherd.BackendClaude, Prompt: quickstartPrompt(document), Root: dir,
		WorkstreamID: workstreamID,
	})
	if err != nil {
		return session, err
	}
	if err := controller.SetSessionTitle(ctx, session.ID, "Quickstart"); err != nil {
		return session, fmt.Errorf("title Quickstart session %s: %w (the session is still available in shepherd)", format.ShortID(session.ID), err)
	}
	return session, nil
}

func quickstartPrompt(document string) string {
	return fmt.Sprintf("You are the Shepherd Quickstart guide. Read %q and follow it exactly. Teach one action at a time and wait for me after each action.", document)
}

func (a *app) runSpawn(args []string) error {
	_, settings, err := a.settings()
	if err != nil {
		return err
	}
	flags := a.newFlagSet("shepherd spawn")
	root := flags.String("root", a.workdir(), "agent working directory")
	flags.StringVar(root, "C", *root, "agent working directory")
	runnerValue := flags.String("runner", string(settings.DefaultRunner), "runner: codex, claude, or no-agent")
	flags.StringVar(runnerValue, "r", *runnerValue, "runner: codex, claude, or no-agent")
	socket := flags.String("socket", defaultSocket(), "private tmux socket name")
	workstreamQuery := flags.String("workstream", "", "workstream name or id (default: Ungrouped)")
	flags.StringVar(workstreamQuery, "w", *workstreamQuery, "workstream name or id (default: Ungrouped)")
	jsonOutput := flags.Bool("json", false, "write a machine-readable result")
	if err := parseAnywhere(flags, args); err != nil {
		return err
	}
	prompt := strings.TrimSpace(strings.Join(flags.Args(), " "))
	if prompt == "" {
		return errors.New("usage: shepherd spawn [-r codex|claude|no-agent] [-C dir] <task-or-label>; put -- before a task that starts with a dash")
	}
	backend, err := shepherd.ParseBackend(*runnerValue)
	if err != nil {
		return err
	}
	controller, err := a.dial(*socket)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	workstreamID := ""
	if strings.TrimSpace(*workstreamQuery) != "" {
		snapshot, snapshotErr := controller.Snapshot(ctx)
		if snapshotErr != nil {
			return snapshotErr
		}
		workstreamID, err = resolveWorkstream(snapshot, *workstreamQuery)
		if err != nil {
			return err
		}
	}
	session, err := controller.Start(ctx, control.StartRequest{
		Backend: backend, Prompt: prompt, Root: *root, WorkstreamID: workstreamID,
	})
	if err != nil {
		return err
	}
	name := "shepherd-" + session.ID
	if session.Runtime != nil {
		name = session.Runtime.Name
	}
	if *jsonOutput {
		return writeJSON(a.out, map[string]any{
			"id": session.ID, "runner": session.Backend, "state": cliStatus(session),
			"workstream_id": session.WorkstreamID, "runtime_name": name, "root": session.Root,
		})
	}
	fmt.Fprintf(a.out, "started %s %s (%s)\n", session.Backend, format.ShortID(session.ID), name)
	return nil
}

func (a *app) runList(args []string) error {
	flags := a.newFlagSet("shepherd list")
	socket := flags.String("socket", defaultSocket(), "private tmux socket name")
	jsonOutput := flags.Bool("json", false, "write a machine-readable snapshot")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("usage: shepherd list")
	}
	controller, err := a.dial(*socket)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	snapshot, err := controller.Snapshot(ctx)
	if err != nil {
		return err
	}
	if *jsonOutput {
		return writeJSON(a.out, newCLISnapshot(snapshot))
	}
	if len(snapshot.Sessions) == 0 && len(snapshot.Orphans) == 0 {
		fmt.Fprintln(a.out, "no shepherd sessions")
		return nil
	}
	writer := tabwriter.NewWriter(a.out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(writer, "ID\tRUNNER\tSTATE\tWORKSTREAM\tRUNTIME\tROOT\tSESSION")
	all := append(append([]control.Session(nil), snapshot.Sessions...), snapshot.Orphans...)
	for _, session := range all {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			format.ShortID(session.ID), session.Backend, cliStatus(session), sessionGroup(snapshot, session),
			format.Duration(session.RuntimeDuration(time.Now())), format.OneLine(format.CompactPath(session.Root)), cliSessionSummary(session))
	}
	return writer.Flush()
}

func (a *app) runSend(args []string) error {
	flags := a.newFlagSet("shepherd send")
	socket := flags.String("socket", defaultSocket(), "private tmux socket name")
	jsonOutput := flags.Bool("json", false, "write a machine-readable result")
	if err := parseAnywhere(flags, args); err != nil {
		return err
	}
	if flags.NArg() < 2 {
		return errors.New("usage: shepherd send <session-id> <message>; put -- before a message that starts with a dash")
	}
	controller, err := a.dial(*socket)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	session, err := controller.Find(ctx, flags.Arg(0))
	if err != nil {
		return err
	}
	if err := controller.Send(ctx, session.ID, strings.Join(flags.Args()[1:], " ")); err != nil {
		return err
	}
	if *jsonOutput {
		return writeJSON(a.out, map[string]any{"session_id": session.ID, "status": "sent"})
	}
	fmt.Fprintln(a.out, "sent to", format.ShortID(session.ID))
	return nil
}

func (a *app) runAttach(args []string) error {
	flags := a.newFlagSet("shepherd attach")
	socket := flags.String("socket", defaultSocket(), "private tmux socket name")
	if err := parseAnywhere(flags, args); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return errors.New("usage: shepherd attach <session-id>")
	}
	controller, err := a.dial(*socket)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	session, err := controller.Find(ctx, flags.Arg(0))
	if err != nil {
		return err
	}
	fmt.Fprintln(a.err, "detach back with Ctrl-\\ or Ctrl-b d")
	fmt.Fprintln(a.err, "drag copies through tmux · Shift-drag selects natively · iTerm2 uses Option")
	command, err := controller.AttachCommand(ctx, session.ID)
	if err != nil {
		return err
	}
	return command.Run()
}

func (a *app) runStop(args []string) error {
	flags := a.newFlagSet("shepherd stop")
	socket := flags.String("socket", defaultSocket(), "private tmux socket name")
	if err := parseAnywhere(flags, args); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return errors.New("usage: shepherd stop <session-id>")
	}
	controller, err := a.dial(*socket)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	session, err := controller.Find(ctx, flags.Arg(0))
	if err != nil {
		return err
	}
	if err := controller.Stop(ctx, session.ID); err != nil {
		return err
	}
	fmt.Fprintln(a.out, "stopped", format.ShortID(session.ID))
	return nil
}

func (a *app) runDoctor(args []string) error {
	configStore, settings, err := a.settings()
	if err != nil {
		return err
	}
	flags := a.newFlagSet("shepherd doctor")
	socket := flags.String("socket", defaultSocket(), "private tmux socket name")
	deep := flags.Bool("deep", false, "validate state, locks, permissions, socket, and clipboard integration")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("usage: shepherd doctor [--deep]")
	}
	stateStore, err := workstream.DefaultStore()
	if err != nil {
		return err
	}
	checks := []struct {
		name     string
		binary   string
		backend  shepherd.Backend
		version  []string
		required bool
		runner   bool
	}{
		{name: "tmux", binary: "tmux", version: []string{"-V"}, required: true},
		{name: "codex", backend: shepherd.BackendCodex, version: []string{"--version"}, runner: true},
		{name: "claude", backend: shepherd.BackendClaude, version: []string{"--version"}, runner: true},
	}
	failed := false
	runnersFound := 0
	for _, check := range checks {
		path := ""
		var command []string
		var err error
		if check.runner {
			command, err = runner.ResolveCommand(check.backend, settings.Command(check.backend))
			if err == nil {
				path = command[0]
			}
		} else {
			path, err = exec.LookPath(check.binary)
			command = []string{path}
		}
		if err != nil {
			label := "optional"
			if check.required {
				label, failed = "required", true
			}
			requested := check.binary
			if check.runner {
				configured := settings.Command(check.backend)
				if len(configured) > 0 {
					requested = configured[0]
				}
			}
			fmt.Fprintf(a.out, "[missing] %-7s %s (%s)\n", check.name, format.OneLine(requested), label)
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		versionArguments := append(append([]string(nil), command[1:]...), check.version...)
		output, err := exec.CommandContext(ctx, path, versionArguments...).CombinedOutput()
		cancel()
		if err != nil {
			fmt.Fprintf(a.out, "[warn]    %-7s %s\n", check.name, format.OneLine(path))
			continue
		}
		if check.runner {
			runnersFound++
		}
		versionText := format.OneLine(string(output))
		if check.name == "tmux" {
			if supported, known := supportedTmuxVersion(versionText); known && !supported {
				failed = true
				fmt.Fprintf(a.out, "[unsupported] %-7s %s · %s (need tmux 3.3+)\n", check.name, format.OneLine(path), versionText)
				continue
			}
		}
		fmt.Fprintf(a.out, "[ok]      %-7s %s · %s\n", check.name, format.OneLine(path), versionText)
	}
	if runnersFound == 0 {
		failed = true
		fmt.Fprintln(a.out, "[missing] runner  no runnable codex or claude installation found")
	}
	fmt.Fprintf(a.out, "[config]  socket  tmux -L %s\n", format.OneLine(*socket))
	fmt.Fprintf(a.out, "[config]  file    %s\n", format.OneLine(configStore.Path))
	fmt.Fprintf(a.out, "[state]   file    %s\n", format.OneLine(stateStore.Path))
	fmt.Fprintf(a.out, "[state]   files   %s\n", format.OneLine(stateStore.Artifacts))
	fmt.Fprintf(a.out, "[config]  runner  %s\n", settings.DefaultRunner)
	fmt.Fprintf(a.out, "[config]  root    %s\n", format.OneLine(a.workdir()))
	if *deep {
		if deepDoctorFailed(a.out, configStore.Path, stateStore, *socket) {
			failed = true
		}
	}
	if failed {
		return errors.New("doctor found problems")
	}
	fmt.Fprintln(a.out, "[next]   tour    shepherd quickstart")
	return nil
}

func deepDoctorFailed(writer io.Writer, configPath string, stateStore workstream.FileStore, socket string) bool {
	failed := false
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	state, err := stateStore.Load(ctx)
	cancel()
	if err != nil {
		failed = true
		fmt.Fprintf(writer, "[fail]    state   %s\n", format.OneLine(err.Error()))
	} else {
		fmt.Fprintf(writer, "[ok]      state   schema %d · revision %d · %d workstreams · %d sessions\n",
			state.Version, state.Revision, len(state.Workstreams), len(state.Sessions))
	}

	lockCtx, lockCancel := context.WithTimeout(context.Background(), 750*time.Millisecond)
	lockErr := stateStore.WithLifecycleLock(lockCtx, func() error { return nil })
	lockCancel()
	if lockErr != nil {
		failed = true
		fmt.Fprintf(writer, "[fail]    lock    %s\n", format.OneLine(lockErr.Error()))
	} else {
		fmt.Fprintln(writer, "[ok]      lock    state and lifecycle locks are available")
	}

	permissionPaths := []doctorPermission{
		{label: "home", path: filepath.Dir(stateStore.Path), directory: true, required: true},
		{label: "config", path: configPath},
		{label: "state", path: stateStore.Path, required: err == nil && state.Revision > 0},
		{label: "state lock", path: stateStore.Path + ".lock", required: true},
		{label: "lifecycle lock", path: stateStore.Path + ".lifecycle.lock", required: true},
		{label: "artifacts", path: stateStore.Artifacts, directory: true, required: err == nil && len(state.Workstreams) > 0},
	}
	if err == nil {
		for _, item := range state.Workstreams {
			permissionPaths = append(permissionPaths, doctorPermission{
				label: "workstream " + format.ShortID(item.ID), path: item.ArtifactDir, directory: true, required: true,
			})
		}
	}
	seen := make(map[string]bool)
	for _, check := range permissionPaths {
		if seen[check.path] {
			continue
		}
		seen[check.path] = true
		if !doctorPrivatePath(writer, check) {
			failed = true
		}
	}

	manager, managerErr := supervisor.New(socket)
	if managerErr != nil {
		failed = true
		fmt.Fprintf(writer, "[fail]    socket  %s\n", format.OneLine(managerErr.Error()))
		return failed
	}
	socketCtx, socketCancel := context.WithTimeout(context.Background(), 3*time.Second)
	health, inspectErr := manager.Inspect(socketCtx)
	socketCancel()
	if inspectErr != nil {
		failed = true
		fmt.Fprintf(writer, "[fail]    socket  %s\n", format.OneLine(inspectErr.Error()))
		return failed
	}
	if !health.ServerRunning {
		fmt.Fprintln(writer, "[ok]      socket  inactive; it will start on the first session launch")
		return failed
	}
	fmt.Fprintf(writer, "[ok]      socket  active · %d sessions\n", health.SessionCount)
	if health.Bootstrap != supervisor.ExpectedBootstrapVersion() {
		failed = true
		fmt.Fprintf(writer, "[fail]    tmux    bootstrap %q, want %q; reopen Shepherd to refresh it\n",
			health.Bootstrap, supervisor.ExpectedBootstrapVersion())
	} else {
		fmt.Fprintf(writer, "[ok]      tmux    bootstrap %s\n", health.Bootstrap)
	}
	if health.Mouse != "on" || !strings.Contains(health.DragBinding, "copy-mode") || !strings.Contains(health.DragBinding, "-M") {
		failed = true
		fmt.Fprintln(writer, "[fail]    mouse   copy-first drag binding is not active")
	} else {
		fmt.Fprintln(writer, "[ok]      mouse   copy-first selection is active")
	}
	clipboardOK := health.SetClipboard == "on" && strings.Contains(health.CopyBinding, "copy-pipe-and-cancel")
	if runtime.GOOS == "darwin" {
		clipboardOK = clipboardOK && health.CopyCommand == "/usr/bin/pbcopy"
	}
	if !clipboardOK {
		failed = true
		fmt.Fprintln(writer, "[fail]    copy    tmux clipboard integration is incomplete")
	} else {
		fmt.Fprintln(writer, "[ok]      copy    tmux clipboard integration is active")
	}
	if len(health.DegradedPanes) > 0 {
		failed = true
		fmt.Fprintf(writer, "[fail]    panes   %d session panes have degraded Shepherd metadata\n", len(health.DegradedPanes))
		for _, pane := range health.DegradedPanes {
			fmt.Fprintf(writer, "[fail]    pane    %s · %s\n", format.ShortID(pane.ID), format.OneLine(pane.Reason))
		}
	} else {
		fmt.Fprintln(writer, "[ok]      panes   all Shepherd pane metadata is readable")
	}
	return failed
}

type doctorPermission struct {
	label     string
	path      string
	directory bool
	required  bool
}

func doctorPrivatePath(writer io.Writer, check doctorPermission) bool {
	info, err := os.Lstat(check.path)
	if errors.Is(err, os.ErrNotExist) {
		if check.required {
			fmt.Fprintf(writer, "[fail]    perms   %-14s missing: %s\n", check.label, format.OneLine(check.path))
			return false
		}
		fmt.Fprintf(writer, "[skip]    perms   %-14s not created yet\n", check.label)
		return true
	}
	if err != nil {
		fmt.Fprintf(writer, "[fail]    perms   %-14s %s\n", check.label, format.OneLine(err.Error()))
		return false
	}
	if info.Mode()&os.ModeSymlink != 0 {
		fmt.Fprintf(writer, "[fail]    perms   %-14s symlink is not accepted: %s\n", check.label, format.OneLine(check.path))
		return false
	}
	if check.directory != info.IsDir() {
		kind := "file"
		if check.directory {
			kind = "directory"
		}
		fmt.Fprintf(writer, "[fail]    perms   %-14s expected %s: %s\n", check.label, kind, format.OneLine(check.path))
		return false
	}
	required := os.FileMode(0o600)
	if check.directory {
		required = 0o700
	}
	mode := info.Mode().Perm()
	if mode&0o077 != 0 || mode&required != required {
		fmt.Fprintf(writer, "[fail]    perms   %-14s %04o (want private owner access)\n", check.label, mode)
		return false
	}
	fmt.Fprintf(writer, "[ok]      perms   %-14s %04o\n", check.label, mode)
	return true
}

func printHelp(writer io.Writer) {
	fmt.Fprintln(writer, `shepherd — a fast dashboard for parallel native coding agents

Usage:
  shepherd [--runner codex|claude|no-agent] [-C DIR]
                                        open the dashboard
  shepherd quickstart                    launch Claude in ~/.shepherd for an agent-guided tour
  shepherd spawn [--json] [-r RUNNER] [-C DIR] [-w WORKSTREAM] LABEL
                                        start a session without the dashboard
  shepherd list [--json]                      list sessions
  shepherd send [--json] ID MESSAGE           send a follow-up through tmux
  shepherd attach ID                          enter the native agent terminal
  shepherd peek ID [--lines N]                print the pane's current frame
  shepherd history ID [--last N] [--json]     print what the runner recorded happened
  shepherd conversation ID [--json]           print the runner conversation id Shepherd registered
  shepherd resume [--json] ID MESSAGE         send to its live Codex owner, or resume when unowned
  shepherd fork [--json] ID MESSAGE           explicitly branch a Codex conversation
  shepherd stop ID                            stop runtime; keep the durable record
  shepherd doctor [--deep]                    check dependencies; deeply validate local runtime health

Organize (the same actions the dashboard chords perform, plus roots and archive):
  shepherd ws list [--json]                   list workstreams, roots, session counts
  shepherd ws create NAME [-C DIR] [-d DESC]  create a workstream; DIR is its first root
  shepherd ws rename WS NAME                  rename a workstream
  shepherd ws reorder WS --up|--down          move it in the dashboard's display order
  shepherd ws archive WS --yes                archive it; members become Ungrouped
  shepherd ws root add WS DIR                 register a launch root
  shepherd ws root set WS OLD NEW             replace a registered root
  shepherd ws root rm WS DIR                  unregister a root; files are untouched
  shepherd title ID TITLE | shepherd title ID --clear
                                        set or clear a durable session title
  shepherd move ID --workstream WS|--ungrouped
                                        change workstream membership
  shepherd reorder ID --up|--down            move it within its workstream
  shepherd adopt ID [-w WORKSTREAM]           claim an orphaned tmux pane
  shepherd delete ID --yes                    delete a durable record with no runtime

Pilot:
  shepherd init [--force]                     write the agent instructions into ~/.shepherd

Workstreams and sessions accept a full id, an id prefix, or a workstream name.

Dashboard:
  Enter             send the draft where the composer prefix says it goes
  Empty Space       aim the composer at the selected live session
  Shift-Enter       insert a composer newline (Ctrl-J fallback)
  Option-←/→        move by word; Option-Delete deletes a word
  Command-←/→       move to logical line start/end
  Command-↑/↓       move to whole-draft start/end
  Tab               switch the new-session runner (default binding)
  Shift-Tab         cycle workstream roots (default binding)
  F1 / Empty ?      open scrollable help and the noun glossary
  Ctrl-S / F2       open settings (e edits JSON, r reloads)
  F3                re-read sessions, preview, and the selected notes/files
  Up / Down         select a session; move draft lines when multiline
  Ctrl-G            resize snapshot/context with Up/Down; r resets
  Empty Enter       collapse a workstream or attach a session (not while replying)
  Ctrl-N            create a workstream, named through the composer
  Ctrl-R            rename a workstream or edit/clear a session title
  Ctrl-T            mark a session; Ctrl-T on a workstream moves or adopts it
  Shift-↑/↓         reorder a workstream or a session within its workstream
  Ctrl-b d          detach the native terminal back to shepherd
  Ctrl-\            alternate one-chord detach shortcut
  Ctrl-X twice      stop runtime; repeat once pane-free to delete record
  Esc               leave a reply, clear the composer, then select Ungrouped
  Ctrl-C            quit the dashboard; Esc never quits

Composer bindings are configurable in JSON and shown in settings/help.
Closing shepherd never stops agents. The s and S aliases invoke the same binary.`)
}

func (a *app) newFlagSet(name string) *flag.FlagSet {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(a.err)
	return flags
}

func defaultSocket() string {
	return env.ValueOr(env.TmuxSocket, supervisor.DefaultSocket)
}

func loadSettings() (config.Store, config.Config, error) {
	store, err := config.DefaultStore()
	if err != nil {
		return config.Store{}, config.Config{}, err
	}
	settings, err := store.Load()
	if err != nil {
		return config.Store{}, config.Config{}, err
	}
	return store, settings, nil
}

func newController(socket string) (*supervisor.Tmux, *control.Controller, workstream.FileStore, error) {
	manager, err := supervisor.New(socket)
	if err != nil {
		return nil, nil, workstream.FileStore{}, err
	}
	stateStore, err := workstream.DefaultStore()
	if err != nil {
		return nil, nil, workstream.FileStore{}, err
	}
	configStore, err := config.DefaultStore()
	if err != nil {
		return nil, nil, workstream.FileStore{}, err
	}
	resolver := control.ResolveCommandFunc(func(_ context.Context, backend shepherd.Backend) ([]string, error) {
		if backend == shepherd.BackendNoAgent {
			return nil, nil
		}
		settings, err := configStore.Load()
		if err != nil {
			return nil, err
		}
		return runner.ResolveCommand(backend, settings.Command(backend))
	})
	return manager, control.New(manager, stateStore, socket,
		control.WithCommandResolver(resolver),
		control.WithConversationResolver(conversationResolver(transcript.Reader{})),
	), stateStore, nil
}

// conversationResolver answers "which conversation did this runner mint for
// this launch" from the files the runner wrote.
//
// Only Codex ever reaches it. Claude takes --session-id, so its conversation is
// registered at launch from what Shepherd passed and no lookup happens; a runner
// that reached here without minting its own id would be answered with a refusal
// rather than a scan, because scanning for something already known is how a
// certainty gets downgraded into a guess.
func conversationResolver(reader transcript.Reader) control.ConversationResolver {
	return control.ResolveConversationFunc(func(_ context.Context, record workstream.SessionRecord) (string, error) {
		if record.Backend != shepherd.BackendCodex {
			return "", fmt.Errorf("runner %s names its own conversation at launch; nothing to resolve", record.Backend)
		}
		found, err := reader.FindConversation(transcript.ConversationRequest{
			Runner:    record.Backend,
			Root:      record.InitialRoot,
			StartedAt: record.CreatedAt,
			Prompt:    record.InitialPrompt,
		})
		if err != nil {
			return "", codexResolveError(err)
		}
		return found.ID, nil
	})
}

// codexResolveError turns a failed match into a sentence that says what Shepherd
// looked for and why it will not answer, rather than reporting a bare miss.
func codexResolveError(err error) error {
	switch {
	case errors.Is(err, transcript.ErrConversationNotFound):
		return fmt.Errorf(
			"codex mints its own conversation id, and no rollout under ~/.codex/sessions matches this "+
				"session's launch directory, start time and initial prompt: %w", err)
	case errors.Is(err, transcript.ErrConversationAmbiguous):
		return fmt.Errorf(
			"more than one codex rollout matches this session's launch directory, start time and initial "+
				"prompt, so Shepherd cannot tell which conversation is this one: %w", err)
	default:
		return err
	}
}

func resolveWorkstream(snapshot control.Snapshot, query string) (string, error) {
	query = strings.TrimSpace(query)
	var matches []workstream.Workstream
	for _, item := range snapshot.Workstreams {
		if item.ID == query || strings.EqualFold(item.Name, query) {
			return item.ID, nil
		}
		if strings.HasPrefix(item.ID, query) || strings.HasPrefix(strings.ToLower(item.Name), strings.ToLower(query)) {
			matches = append(matches, item)
		}
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no active workstream matches %q", query)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("workstream query %q is ambiguous", query)
	}
	return matches[0].ID, nil
}

type cliSnapshotJSON struct {
	Revision    uint64              `json:"revision"`
	Workstreams []cliWorkstreamJSON `json:"workstreams"`
	Sessions    []cliSessionJSON    `json:"sessions"`
}

type cliWorkstreamJSON struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	ArtifactDir string   `json:"artifact_dir"`
	Roots       []string `json:"roots"`
	Revision    uint64   `json:"revision"`
}

type cliSessionJSON struct {
	ID                string           `json:"id"`
	Runner            shepherd.Backend `json:"runner"`
	State             string           `json:"state"`
	Title             string           `json:"title,omitempty"`
	DisplayTitle      string           `json:"display_title"`
	InitialPrompt     string           `json:"initial_prompt"`
	LatestViaShepherd string           `json:"latest_via_shepherd,omitempty"`
	WorkstreamID      string           `json:"workstream_id,omitempty"`
	Workstream        string           `json:"workstream"`
	Root              string           `json:"root"`
	Available         bool             `json:"available"`
	Alive             bool             `json:"alive"`
	Orphaned          bool             `json:"orphaned"`
	ExitCode          *int             `json:"exit_code"`
	RuntimeSeconds    int64            `json:"runtime_seconds"`
	LastActivityAt    *time.Time       `json:"last_activity_at,omitempty"`
	NativeStatus      string           `json:"native_status,omitempty"`
	DegradedReason    string           `json:"degraded_reason,omitempty"`
}

func newCLISnapshot(snapshot control.Snapshot) cliSnapshotJSON {
	result := cliSnapshotJSON{
		Revision:    snapshot.Revision,
		Workstreams: make([]cliWorkstreamJSON, 0, len(snapshot.Workstreams)),
		Sessions:    make([]cliSessionJSON, 0, len(snapshot.Sessions)+len(snapshot.Orphans)),
	}
	for _, item := range snapshot.Workstreams {
		result.Workstreams = append(result.Workstreams, cliWorkstreamJSON{
			ID: item.ID, Name: item.Name, Description: item.Description,
			ArtifactDir: item.ArtifactDir, Roots: append([]string(nil), item.Roots...), Revision: item.Revision,
		})
	}
	all := append(append([]control.Session(nil), snapshot.Sessions...), snapshot.Orphans...)
	for _, session := range all {
		var exitCode *int
		if code, known := session.ExitCode(); known {
			value := code
			exitCode = &value
		}
		title := strings.TrimSpace(session.Record.Title)
		displayTitle := title
		if displayTitle == "" {
			displayTitle = format.OneLine(session.Prompt)
		}
		var lastActivityAt *time.Time
		if observed := session.LastActivity(); !observed.IsZero() {
			value := observed
			lastActivityAt = &value
		}
		nativeStatus := ""
		degradedReason := ""
		if session.Runtime != nil {
			nativeStatus = session.Runtime.NativeStatus
			degradedReason = session.Runtime.ObservationError
		}
		result.Sessions = append(result.Sessions, cliSessionJSON{
			ID: session.ID, Runner: session.Backend, State: string(session.Status),
			Title: title, DisplayTitle: displayTitle, InitialPrompt: session.Prompt,
			LatestViaShepherd: session.LastUserMessage,
			WorkstreamID:      session.WorkstreamID, Workstream: sessionGroup(snapshot, session),
			Root: session.Root, Available: session.Available(), Alive: session.Alive(), Orphaned: session.Orphaned,
			ExitCode: exitCode, RuntimeSeconds: int64(session.RuntimeDuration(time.Now()).Seconds()),
			LastActivityAt: lastActivityAt,
			NativeStatus:   nativeStatus,
			DegradedReason: degradedReason,
		})
	}
	return result
}

func writeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return fmt.Errorf("write JSON: %w", err)
	}
	return nil
}

func cliStatus(session control.Session) string {
	if session.Status == control.StatusExited {
		if code, ok := session.ExitCode(); ok {
			if code != 0 {
				return fmt.Sprintf("failed(%d)", code)
			}
		} else {
			return "exited(?)"
		}
	}
	return string(session.Status)
}

func cliSessionSummary(session control.Session) string {
	title := strings.TrimSpace(session.Record.Title)
	if title == "" {
		title = session.Prompt
	}
	title = format.OneLine(title)
	if latest := format.OneLine(session.LastUserMessage); latest != "" && latest != title {
		return title + " · latest: " + latest
	}
	return title
}

func sessionGroup(snapshot control.Snapshot, session control.Session) string {
	if session.Orphaned {
		return "Orphaned"
	}
	if session.WorkstreamID == "" {
		return "Ungrouped"
	}
	for _, item := range snapshot.Workstreams {
		if item.ID == session.WorkstreamID {
			return item.Name
		}
	}
	return "Unavailable"
}

func mustWorkingDirectory() string {
	value, err := os.Getwd()
	if err != nil {
		return "."
	}
	return value
}

func supportedTmuxVersion(value string) (supported, known bool) {
	fields := strings.Fields(value)
	if len(fields) < 2 || fields[0] != "tmux" {
		return false, false
	}
	parts := strings.SplitN(fields[1], ".", 2)
	if len(parts) != 2 {
		return false, false
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return false, false
	}
	minorDigits := strings.TrimRightFunc(parts[1], func(r rune) bool { return r < '0' || r > '9' })
	if minorDigits == "" {
		return false, false
	}
	minor, err := strconv.Atoi(minorDigits)
	if err != nil {
		return false, false
	}
	return major > 3 || (major == 3 && minor >= 3), true
}
