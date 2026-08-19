#!/bin/sh
# Exercise the exact local install shape plus a fresh home and the durable
# schema shipped by 0.7.9. This deliberately uses no real runner credentials.
set -eu

root="$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)"
scratch="$(mktemp -d "${TMPDIR:-/tmp}/shepherd-release-smoke.XXXXXX")"
socket="shepherd-release-smoke-$$"
cleanup() {
	tmux -L "$socket" kill-server >/dev/null 2>&1 || true
	rm -rf "$scratch"
}
trap cleanup EXIT HUP INT TERM

prefix="$scratch/prefix"
user_home="$scratch/user"
fresh_home="$scratch/fresh"
upgrade_home="$scratch/upgrade"
mkdir -m 700 "$user_home" "$fresh_home" "$upgrade_home"

make -s -C "$root" install PREFIX="$prefix" >/dev/null
test -x "$prefix/bin/shepherd"
test -L "$prefix/bin/s"
test -L "$prefix/bin/S"

source_version="$(sed -n 's/^var version = "\([^"]*\)"$/\1/p' "$root/cmd/shepherd/main.go")"
reported_version="$("$prefix/bin/shepherd" --version)"
test "$reported_version" = "shepherd $source_version"

fake_codex="$scratch/fake-codex"
printf '%s\n' '#!/bin/sh' 'echo "codex-smoke 0.0.0"' >"$fake_codex"
chmod 700 "$fake_codex"

HOME="$user_home" \
	SHEPHERD_HOME="$fresh_home" \
	SHEPHERD_CODEX_BIN="$fake_codex" \
	SHEPHERD_TMUX_SOCKET="$socket" \
	"$prefix/bin/shepherd" doctor --deep >/dev/null
test -f "$fresh_home/QUICKSTART.md"
cmp -s "$fresh_home/QUICKSTART.md" "$root/skills/learn-shepherd/SKILL.md"

upgrade_artifacts="$upgrade_home/workstreams"
upgrade_root="$scratch/upgrade-root"
mkdir -m 700 "$upgrade_artifacts" "$upgrade_artifacts/018f0000-0000-4000-8000-000000000400" "$upgrade_root"
sed \
	-e "s#/tmp/shepherd-v4-artifacts#$upgrade_artifacts#g" \
	-e "s#/tmp/shepherd-v4-root#$upgrade_root#g" \
	"$root/internal/workstream/testdata/state-v4-valid.json" >"$upgrade_home/state.json"
chmod 600 "$upgrade_home/state.json"
upgrade_json="$(
	HOME="$user_home" \
		SHEPHERD_HOME="$upgrade_home" \
		SHEPHERD_CODEX_BIN="$fake_codex" \
		SHEPHERD_TMUX_SOCKET="$socket" \
		"$prefix/bin/shepherd" list --json
)"
case "$upgrade_json" in
	*"Keep this one first"*"Keep this one second"*) ;;
	*)
		echo "release smoke: 0.7.9 session ordering did not survive upgrade" >&2
		exit 1
		;;
esac

HOME="$user_home" \
	SHEPHERD_HOME="$upgrade_home" \
	SHEPHERD_CODEX_BIN="$fake_codex" \
	SHEPHERD_TMUX_SOCKET="$socket" \
	"$prefix/bin/shepherd" doctor --deep >/dev/null

printf 'release-smoke: install, fresh home, and 0.7.9 state passed for %s\n' "$reported_version"
