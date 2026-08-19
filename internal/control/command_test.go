package control

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ez-gz/shepherd/internal/shepherd"
)

func TestHumanWrappersUseCommandAuthorizationBoundary(t *testing.T) {
	repository := newMemoryRepository(t.TempDir())
	want := errors.New("policy denied")
	var observed Command
	controller := New(&fakeSupervisor{}, repository, "shepherd-test", WithAuthorizer(AuthorizeFunc(
		func(_ context.Context, command Command) error {
			observed = command
			return want
		},
	)))

	_, err := controller.CreateWorkstream(context.Background(), "Core", "", []string{t.TempDir()})
	if !errors.Is(err, want) {
		t.Fatalf("CreateWorkstream error = %v, want policy error", err)
	}
	if observed.Actor != LocalHuman() || observed.Scope != InstallationScope() {
		t.Fatalf("observed command actor/scope = %#v/%#v", observed.Actor, observed.Scope)
	}
	if _, ok := observed.Action.(CreateWorkstreamAction); !ok {
		t.Fatalf("observed action = %T, want CreateWorkstreamAction", observed.Action)
	}
	state, loadErr := repository.Load(context.Background())
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	if len(state.Workstreams) != 0 {
		t.Fatalf("denied command mutated state: %#v", state.Workstreams)
	}
}

// The dashboard's archive chord calls ArchiveWorkstream, so this is the seam
// that decides whether a keystroke reaches durable state as a typed,
// scoped, authorized command or as a bare store write.
func TestArchiveWorkstreamCrossesTheCommandBoundaryScopedToItsWorkstream(t *testing.T) {
	root := t.TempDir()
	repository := newMemoryRepository(root)
	var observed Command
	controller := New(&fakeSupervisor{}, repository, "shepherd-test", WithAuthorizer(AuthorizeFunc(
		func(_ context.Context, command Command) error {
			observed = command
			return nil
		},
	)))
	container, err := controller.CreateWorkstream(context.Background(), "Core", "", []string{root})
	if err != nil {
		t.Fatal(err)
	}
	if err := controller.ArchiveWorkstream(context.Background(), container.ID); err != nil {
		t.Fatal(err)
	}
	if observed.Actor != LocalHuman() || observed.Scope != WorkstreamScope(container.ID) {
		t.Fatalf("observed command actor/scope = %#v/%#v", observed.Actor, observed.Scope)
	}
	if _, ok := observed.Action.(ArchiveWorkstreamAction); !ok {
		t.Fatalf("observed action = %T, want ArchiveWorkstreamAction", observed.Action)
	}

	// An installation-scoped archive is refused before any authorizer runs, so
	// the workstream a caller names cannot be left implicit.
	if _, err := controller.Execute(context.Background(), humanCommand(InstallationScope(), ArchiveWorkstreamAction{})); err == nil ||
		!strings.Contains(err.Error(), "requires workstream scope") {
		t.Fatalf("unscoped archive error = %v", err)
	}
}

func TestSessionActorUsesTypedScopeAndTrustedCommandResolver(t *testing.T) {
	root := t.TempDir()
	repository := newMemoryRepository(root)
	var observed Command
	var launched shepherd.StartRequest
	resolved := []string{"/trusted/codex", "--fixed-flag"}
	supervisor := &fakeSupervisor{}
	controller := New(supervisor, repository, "shepherd-test",
		WithAuthorizer(AuthorizeFunc(func(_ context.Context, command Command) error {
			observed = command
			return nil
		})),
		WithCommandResolver(ResolveCommandFunc(func(_ context.Context, backend shepherd.Backend) ([]string, error) {
			if backend != shepherd.BackendCodex {
				t.Fatalf("resolver backend = %q", backend)
			}
			return append([]string(nil), resolved...), nil
		})),
	)
	container, err := controller.CreateWorkstream(context.Background(), "Core", "", []string{root})
	if err != nil {
		t.Fatal(err)
	}
	actorID := "018f0000-0000-4000-8000-000000000071"
	supervisor.start = func(request shepherd.StartRequest) (shepherd.Session, error) {
		launched = request
		return shepherd.Session{
			ID: request.ID, Name: "shepherd-" + request.ID, Backend: request.Backend,
			Prompt: request.Prompt, Root: request.Root, Status: shepherd.StatusLive,
			StartedAt: time.Now(),
		}, nil
	}

	result, err := controller.Execute(context.Background(), Command{
		Actor:  Actor{Kind: ActorSession, SessionID: actorID},
		Scope:  WorkstreamScope(container.ID),
		Action: StartAction{Backend: shepherd.BackendCodex, Prompt: "build it", Root: root},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Session.ID == "" || observed.Actor.SessionID != actorID || observed.Scope.WorkstreamID != container.ID {
		t.Fatalf("command/result lost identity: command=%#v result=%#v", observed, result)
	}
	if !slices.Equal(launched.Command, resolved) {
		t.Fatalf("supervisor command = %#v, want trusted %#v", launched.Command, resolved)
	}
}

func TestDefaultPolicyRejectsSessionActor(t *testing.T) {
	controller := New(&fakeSupervisor{}, newMemoryRepository(t.TempDir()), "shepherd-test")
	_, err := controller.Execute(context.Background(), Command{
		Actor:  Actor{Kind: ActorSession, SessionID: "018f0000-0000-4000-8000-000000000072"},
		Scope:  InstallationScope(),
		Action: SendAction{SessionID: "018f0000-0000-4000-8000-000000000073", Message: "hello"},
	})
	if err == nil || !strings.Contains(err.Error(), "manager grant") {
		t.Fatalf("session actor error = %v, want manager-grant denial", err)
	}
}

func TestCommandScopeValidationPrecedesAuthorization(t *testing.T) {
	called := false
	controller := New(&fakeSupervisor{}, newMemoryRepository(t.TempDir()), "shepherd-test", WithAuthorizer(AuthorizeFunc(
		func(context.Context, Command) error {
			called = true
			return nil
		},
	)))
	workstreamID := "018f0000-0000-4000-8000-000000000074"
	_, err := controller.Execute(context.Background(), humanCommand(InstallationScope(), RenameWorkstreamAction{Name: "wrong scope"}))
	if err == nil || !strings.Contains(err.Error(), "requires workstream scope") {
		t.Fatalf("scope error = %v", err)
	}
	if called {
		t.Fatal("authorizer ran for structurally invalid command")
	}

	_, err = controller.Execute(context.Background(), humanCommand(InstallationScope(), ReorderSessionAction{
		SessionID: "018f0000-0000-4000-8000-000000000075", Delta: -1,
	}))
	if err == nil || !strings.Contains(err.Error(), "requires workstream scope") {
		t.Fatalf("session reorder scope error = %v", err)
	}

	_, err = controller.Execute(context.Background(), humanCommand(WorkstreamScope(workstreamID), MoveSessionAction{
		SessionID: "018f0000-0000-4000-8000-000000000075", WorkstreamID: "018f0000-0000-4000-8000-000000000076",
	}))
	if err == nil || !strings.Contains(err.Error(), "must match") {
		t.Fatalf("mismatched destination error = %v", err)
	}
}

func TestCommandRejectsOversizedInteractivePayloads(t *testing.T) {
	tooLarge := strings.Repeat("x", shepherd.MaxInteractivePayloadBytes+1)
	tests := []Action{
		StartAction{Prompt: tooLarge},
		ResumeSessionAction{Prompt: tooLarge},
		ForkSessionAction{Prompt: tooLarge},
		SendAction{Message: tooLarge},
	}
	for _, action := range tests {
		command := humanCommand(InstallationScope(), action)
		if err := validateCommand(command); err == nil || !strings.Contains(err.Error(), "maximum") {
			t.Errorf("validateCommand(%T) error = %v", action, err)
		}
	}
}
