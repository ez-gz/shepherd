package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/ez-gz/shepherd/internal/control"
	"github.com/ez-gz/shepherd/internal/control/controltest"
	"github.com/ez-gz/shepherd/internal/format"
	"github.com/ez-gz/shepherd/internal/shepherd"
	"github.com/ez-gz/shepherd/internal/workstream"
)

func TestRouteGlobalCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		handled bool
		want    string
	}{
		{name: "help", args: []string{"help"}, handled: true, want: "shepherd — a fast dashboard"},
		{name: "short help", args: []string{"-h"}, handled: true, want: "shepherd — a fast dashboard"},
		{name: "long help", args: []string{"--help"}, handled: true, want: "shepherd — a fast dashboard"},
		{name: "version", args: []string{"version"}, handled: true, want: "shepherd " + version + "\n"},
		{name: "long version", args: []string{"--version"}, handled: true, want: "shepherd " + version + "\n"},
		{name: "dashboard", handled: false},
		{name: "dashboard flag", args: []string{"--runner", "no-agent"}, handled: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			handled := routeGlobalCommand(test.args, &output)
			if handled != test.handled {
				t.Fatalf("routeGlobalCommand() handled = %v, want %v", handled, test.handled)
			}
			if test.want == "" {
				if output.Len() != 0 {
					t.Fatalf("routeGlobalCommand() output = %q, want empty", output.String())
				}
				return
			}
			if !strings.Contains(output.String(), test.want) {
				t.Fatalf("routeGlobalCommand() output = %q, want substring %q", output.String(), test.want)
			}
		})
	}
}

func TestRunRoutesLongVersionBeforeDashboardFlags(t *testing.T) {
	var output bytes.Buffer
	application := &app{out: &output, err: &output}
	if err := application.run([]string{"--version"}); err != nil {
		t.Fatal(err)
	}
	if got, want := output.String(), "shepherd "+version+"\n"; got != want {
		t.Fatalf("run(--version) = %q, want %q", got, want)
	}
}

// The install command has to stay self-maintaining. @latest resolves to the
// newest release tag, so a release needs no edit here; a pinned version would
// quietly advertise an old release from the moment the next tag is pushed, and
// nothing about the README would look wrong.
func TestReadmeInstallsTheLatestRelease(t *testing.T) {
	readme, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatal(err)
	}
	if want := "go install github.com/ez-gz/shepherd/cmd/shepherd@latest"; !strings.Contains(string(readme), want) {
		t.Fatalf("README install command is not %q", want)
	}
}

// The version this binary reports is what a tag is checked against at release
// time by scripts/check-version.sh. A value that is not semver cannot match any
// tag, and the failure would land on a user running go install rather than here.
func TestVersionIsSemver(t *testing.T) {
	if !regexp.MustCompile(`^\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$`).MatchString(version) {
		t.Fatalf("version %q is not semver", version)
	}
}

func TestHelpAdvertisesQuickstart(t *testing.T) {
	var output bytes.Buffer
	printHelp(&output)
	for _, want := range []string{
		"shepherd quickstart                    launch Claude in ~/.shepherd",
		"shepherd list [--json]",
		"shepherd doctor [--deep]",
		"Ctrl-G            resize snapshot/context",
		"Ctrl-R            rename a workstream or edit/clear a session title",
		"Ctrl-T            mark a session; Ctrl-T on a workstream moves or adopts it",
		"Shift-↑/↓         reorder a workstream or a session within its workstream",
		"Ctrl-C            quit the dashboard; Esc never quits",
	} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("help does not advertise %q:\n%s", want, output.String())
		}
	}
}

func TestQuickstartPromptPointsAtInstalledDocument(t *testing.T) {
	document := "/tmp/shepherd home/QUICKSTART.md"
	prompt := quickstartPrompt(document)
	if !strings.Contains(prompt, document) {
		t.Fatalf("quickstart prompt %q does not point at %q", prompt, document)
	}
	if len(prompt) >= 512 {
		t.Fatalf("quickstart prompt is %d bytes; it should remain a small file pointer", len(prompt))
	}
}

func TestQuickstartHelpReturnsSuccess(t *testing.T) {
	var output bytes.Buffer
	application := &app{out: &output, err: &output, workdir: func() string { return t.TempDir() }}
	if err := application.runQuickstart([]string{"-h"}); err != nil {
		t.Fatalf("quickstart help: %v", err)
	}
}

func TestLaunchQuickstartSeedsFirstInstallBeforeStartingTitledClaude(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".shepherd")
	t.Setenv("SHEPHERD_HOME", dir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	store := workstream.FileStore{
		Path: filepath.Join(dir, "state.json"), Artifacts: filepath.Join(dir, "workstreams"),
	}
	now := time.Now()
	manager := workstream.Workstream{
		ID: testWorkstreamID, Name: ManagersWorkstreamName, Description: managersWorkstreamDescription,
		ArtifactDir: filepath.Join(store.Artifacts, testWorkstreamID), Roots: []string{dir},
		Revision: 1, CreatedAt: now, UpdatedAt: now,
	}
	var events []string
	var workstreams []workstream.Workstream
	var started control.StartRequest
	var title string
	service := &controltest.Stub{
		CreateWorkstreamFunc: func(_ context.Context, name, description string, roots []string) (workstream.Workstream, error) {
			events = append(events, "create-workstream")
			if name != ManagersWorkstreamName || description != managersWorkstreamDescription || len(roots) != 1 || roots[0] != dir {
				t.Fatalf("manager workstream = %q, %q, %v", name, description, roots)
			}
			workstreams = append(workstreams, manager)
			if err := os.WriteFile(store.Path, []byte("first durable write"), 0o600); err != nil {
				t.Fatal(err)
			}
			return manager, nil
		},
		SnapshotFunc: func(context.Context) (control.Snapshot, error) {
			events = append(events, "snapshot")
			return control.Snapshot{Workstreams: workstreams}, nil
		},
		StartFunc: func(_ context.Context, request control.StartRequest) (control.Session, error) {
			events = append(events, "start")
			started = request
			return control.Session{ID: testSessionID}, nil
		},
		SetSessionTitleFunc: func(_ context.Context, id, value string) error {
			events = append(events, "title")
			if id != testSessionID {
				t.Fatalf("titled session %q, want %q", id, testSessionID)
			}
			title = value
			return nil
		},
	}

	if _, err := launchQuickstart(context.Background(), service, store, dir, io.Discard); err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(events, ","), "create-workstream,snapshot,start,title"; got != want {
		t.Fatalf("quickstart events = %q, want %q", got, want)
	}
	if started.Backend != shepherd.BackendClaude || started.Root != dir || started.WorkstreamID != testWorkstreamID {
		t.Fatalf("quickstart request = %+v", started)
	}
	if !strings.Contains(started.Prompt, filepath.Join(dir, quickstartDocumentName)) || len(started.Prompt) >= 512 {
		t.Fatalf("quickstart prompt = %q", started.Prompt)
	}
	if title != "Quickstart" {
		t.Fatalf("quickstart title = %q", title)
	}
}

func TestLaunchQuickstartDoesNotRecreateManagersWorkstreamAfterFirstStartup(t *testing.T) {
	dir := t.TempDir()
	store := workstream.FileStore{Path: filepath.Join(dir, "state.json"), Artifacts: filepath.Join(dir, "workstreams")}
	if err := os.WriteFile(store.Path, []byte("existing durable state"), 0o600); err != nil {
		t.Fatal(err)
	}
	created := false
	service := &controltest.Stub{
		CreateWorkstreamFunc: func(context.Context, string, string, []string) (workstream.Workstream, error) {
			created = true
			return workstream.Workstream{}, nil
		},
		StartFunc: func(_ context.Context, request control.StartRequest) (control.Session, error) {
			if request.WorkstreamID != "" {
				t.Fatalf("quickstart joined unexpected workstream %q", request.WorkstreamID)
			}
			return control.Session{ID: testSessionID}, nil
		},
	}
	if _, err := launchQuickstart(context.Background(), service, store, dir, io.Discard); err != nil {
		t.Fatal(err)
	}
	if created {
		t.Fatal("quickstart recreated the managers workstream after first startup")
	}
}

func TestSupportedTmuxVersion(t *testing.T) {
	tests := []struct {
		value     string
		supported bool
		known     bool
	}{
		{"tmux 3.2a", false, true},
		{"tmux 3.3", true, true},
		{"tmux 3.6a", true, true},
		{"tmux 4.0", true, true},
		{"unknown", false, false},
	}
	for _, test := range tests {
		supported, known := supportedTmuxVersion(test.value)
		if supported != test.supported || known != test.known {
			t.Errorf("supportedTmuxVersion(%q) = (%v, %v), want (%v, %v)",
				test.value, supported, known, test.supported, test.known)
		}
	}
}

func TestDeepDoctorAcceptsFreshPrivateInstallWithoutRunningServer(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux is not installed")
	}
	root := filepath.Join(t.TempDir(), "shepherd-home")
	store := workstream.FileStore{
		Path: filepath.Join(root, "state.json"), Artifacts: filepath.Join(root, "workstreams"),
	}
	var output bytes.Buffer
	if failed := deepDoctorFailed(&output, filepath.Join(root, "config.json"), store, "shepherd-doctor-fresh-test"); failed {
		t.Fatalf("fresh deep doctor failed:\n%s", output.String())
	}
	for _, want := range []string{"[ok]      state", "[ok]      lock", "socket  inactive"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("deep doctor output lacks %q:\n%s", want, output.String())
		}
	}
}

func TestDoctorPrivatePathRejectsBroadPermissionsAndSymlinks(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "state.json")
	if err := os.WriteFile(path, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	check := doctorPermission{label: "state", path: path}
	if !doctorPrivatePath(&output, check) {
		t.Fatalf("private file rejected: %s", output.String())
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if doctorPrivatePath(io.Discard, check) {
		t.Fatal("world-readable state file accepted")
	}
	link := filepath.Join(root, "linked-state.json")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if doctorPrivatePath(io.Discard, doctorPermission{label: "state", path: link}) {
		t.Fatal("state symlink accepted")
	}
}

func TestResolveWorkstreamByNameAndRejectAmbiguity(t *testing.T) {
	snapshot := control.Snapshot{Workstreams: []workstream.Workstream{
		{ID: "018f0000-0000-4000-8000-000000000001", Name: "Shepherd Core"},
		{ID: "018f0000-0000-4000-8000-000000000002", Name: "Shepherd Docs"},
	}}
	id, err := resolveWorkstream(snapshot, "shepherd core")
	if err != nil || id != snapshot.Workstreams[0].ID {
		t.Fatalf("exact name resolution = (%q, %v)", id, err)
	}
	if _, err := resolveWorkstream(snapshot, "she"); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("ambiguous prefix error = %v", err)
	}
}

func TestOneLineStripsTerminalControlSequences(t *testing.T) {
	got := format.OneLine("safe\x1b]52;c;c2VjcmV0\x07 text\nnext\x1b[31m red\x1b[0m")
	if strings.Contains(got, "\x1b") || strings.Contains(got, "c2VjcmV0") {
		t.Fatalf("oneLine retained terminal control payload: %q", got)
	}
	if got != "safe text next red" {
		t.Fatalf("oneLine = %q", got)
	}
}

func TestMachineSnapshotIncludesDurableTitleAndHonestUnknownExit(t *testing.T) {
	id := "018f0000-0000-4000-8000-000000000081"
	workstreamID := "018f0000-0000-4000-8000-000000000082"
	runtime := shepherd.Session{
		ID: id, Name: "shepherd-" + id, Backend: shepherd.BackendCodex,
		Status: shepherd.StatusExited, ExitCode: nil, StartedAt: time.Now().Add(-time.Minute),
		NativeStatus: "Done · gpt-5.6-codex · 42% ctx",
	}
	snapshot := control.Snapshot{
		Revision:    9,
		Workstreams: []workstream.Workstream{{ID: workstreamID, Name: "Release", Roots: []string{"/tmp"}}},
		Sessions: []control.Session{{
			ID: id, Backend: shepherd.BackendCodex, Prompt: "initial task", LastUserMessage: "latest follow-up",
			Root: "/tmp", WorkstreamID: workstreamID, Status: control.StatusExited, Durable: true,
			Record: workstream.SessionRecord{ID: id, Title: "Ship macOS build"}, Runtime: &runtime,
		}},
	}

	machine := newCLISnapshot(snapshot)
	if len(machine.Sessions) != 1 {
		t.Fatalf("machine sessions = %#v", machine.Sessions)
	}
	session := machine.Sessions[0]
	if session.Title != "Ship macOS build" || session.DisplayTitle != session.Title || session.LatestViaShepherd != "latest follow-up" {
		t.Fatalf("machine title projection = %#v", session)
	}
	if session.ExitCode != nil || session.State != "exited" {
		t.Fatalf("unknown exit projection = stable state %q code %#v", session.State, session.ExitCode)
	}
	if session.NativeStatus != runtime.NativeStatus {
		t.Fatalf("native status = %q, want %q", session.NativeStatus, runtime.NativeStatus)
	}

	var output bytes.Buffer
	if err := writeJSON(&output, machine); err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("machine JSON is invalid: %v\n%s", err, output.String())
	}
	if decoded["revision"] != float64(9) {
		t.Fatalf("decoded revision = %#v", decoded["revision"])
	}
	sessions, ok := decoded["sessions"].([]any)
	if !ok || len(sessions) != 1 {
		t.Fatalf("decoded sessions = %#v", decoded["sessions"])
	}
	decodedSession, ok := sessions[0].(map[string]any)
	if !ok {
		t.Fatalf("decoded session = %#v", sessions[0])
	}
	if exitCode, exists := decodedSession["exit_code"]; !exists || exitCode != nil {
		t.Fatalf("unknown exit_code = %#v (present %v), want explicit null", exitCode, exists)
	}
}

func TestMachineSnapshotIncludesDegradedReason(t *testing.T) {
	runtime := shepherd.Session{
		ID: "damaged-runtime", Name: "damaged-runtime", Status: shepherd.StatusLive,
		ObservationError: "invalid Shepherd runtime identity metadata",
	}
	machine := newCLISnapshot(control.Snapshot{Orphans: []control.Session{{
		ID: runtime.ID, Status: control.StatusDegraded, Orphaned: true, Runtime: &runtime,
	}}})
	if len(machine.Sessions) != 1 || machine.Sessions[0].State != "degraded" ||
		machine.Sessions[0].DegradedReason != runtime.ObservationError {
		t.Fatalf("degraded machine projection = %#v", machine.Sessions)
	}
}

func TestMachineSnapshotKeepsStateStableWhenExitCodeIsNonzero(t *testing.T) {
	code := 7
	id := "018f0000-0000-4000-8000-000000000083"
	runtime := shepherd.Session{
		ID: id, Backend: shepherd.BackendCodex, Status: shepherd.StatusFailed,
		ExitCode: &code, StartedAt: time.Now().Add(-time.Minute),
	}
	machine := newCLISnapshot(control.Snapshot{Sessions: []control.Session{{
		ID: id, Backend: shepherd.BackendCodex, Prompt: "failing task", Root: "/tmp",
		Status: control.StatusExited, Durable: true, Runtime: &runtime,
	}}})
	if len(machine.Sessions) != 1 {
		t.Fatalf("machine sessions = %#v", machine.Sessions)
	}
	session := machine.Sessions[0]
	if session.State != "exited" || session.ExitCode == nil || *session.ExitCode != 7 {
		t.Fatalf("failed process projection = state %q code %#v", session.State, session.ExitCode)
	}
}
