#!/bin/sh

# Integration tests for scripts/install.sh.

set -e -f -u

command_name="$(basename "$0")"
download_fail_env="${AGH_INSTALL_TEST_DOWNLOAD_FAIL:-0}"
event_log_env="${AGH_INSTALL_TEST_EVENT_LOG:-}"
test_script_env="${AGH_INSTALL_TEST_SCRIPT:-}"
readonly command_name download_fail_env event_log_env test_script_env

case "$command_name" in
'AdGuardHome')
	printf '%s\n' "$2" >>"$event_log_env"

	exit 0
	;;
'curl')
	while [ "$#" -gt '0' ]; do
		if [ "$1" = '-o' ]; then
			output="$2"
			shift 2

			continue
		fi

		shift
	done

	printf '%s\n' 'download' >>"$event_log_env"
	if [ "$download_fail_env" -eq '1' ]; then
		exit 1
	fi

	: >"$output"

	exit 0
	;;
'id')
	printf '%s\n' '0'

	exit 0
	;;
'tar')
	while [ "$#" -gt '0' ]; do
		if [ "$1" = '-C' ]; then
			output_dir="$2"
			shift 2

			continue
		elif [ "$1" = '-f' ]; then
			archive="$2"
			shift 2

			continue
		fi

		shift
	done

	if ! [ -f "$archive" ]; then
		printf '%s\n' "install-test: archive disappeared: $archive" 1>&2

		exit 1
	fi

	mkdir -p "$output_dir/AdGuardHome"
	ln -s "$test_script_env" "$output_dir/AdGuardHome/AdGuardHome"
	printf '%s\n' 'unpack' >>"$event_log_env"

	exit 0
	;;
'install-test.sh') ;;
*)
	printf '%s\n' "install-test: unexpected command name: $command_name" 1>&2

	exit 1
	;;
esac

script_dir="$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)"
repo_dir="$(CDPATH='' cd -- "$script_dir/../.." && pwd)"
test_dir="$(mktemp -d "${TMPDIR:-/tmp}/agh-install-test.XXXXXX")"
readonly script_dir repo_dir test_dir

cleanup() {
	rm -r "$test_dir"
}
trap cleanup 0 1 2 3 15

fake_bin="$test_dir/bin"
readonly fake_bin

mkdir -p "$fake_bin"
for command in curl id tar; do
	ln -s "$script_dir/install-test.sh" "$fake_bin/$command"
done

prepare_existing() {
	prepare_install_dir="$1"
	prepare_event_log="$2"

	mkdir -p "$prepare_install_dir/AdGuardHome"
	ln -s "$script_dir/install-test.sh" "$prepare_install_dir/AdGuardHome/AdGuardHome"
	: >"$prepare_event_log"
}

run_installer_from() (
	run_cwd="$1"
	run_install_dir="$2"
	run_event_log="$3"
	run_download_fail="$4"
	shift 4

	cd "$run_cwd"

	AGH_INSTALL_TEST_DOWNLOAD_FAIL="$run_download_fail" \
		AGH_INSTALL_TEST_EVENT_LOG="$run_event_log" \
		AGH_INSTALL_TEST_SCRIPT="$script_dir/install-test.sh" \
		PATH="$fake_bin:/usr/bin:/bin" \
		/bin/sh "$repo_dir/scripts/install.sh" \
		-C amd64 -O linux -o "$run_install_dir" "$@"
)

run_installer() {
	run_installer_from "$test_dir" "$@"
}

assert_events() {
	assert_want="$1"
	assert_event_log="$2"
	assert_message="$3"
	assert_got="$(cat "$assert_event_log")"

	if [ "$assert_got" != "$assert_want" ]; then
		printf '%s\n' \
			"install-test: $assert_message" \
			'install-test: observed events:' 1>&2
		sed 's/^/  /' "$assert_event_log" 1>&2

		exit 1
	fi
}

install_dir="$test_dir/install"
event_log="$test_dir/events"
readonly install_dir event_log

prepare_existing "$install_dir" "$event_log"
run_installer "$install_dir" "$event_log" '0' -r
assert_events 'download
stop
uninstall
unpack
install' "$event_log" \
	'package download must finish before stopping the existing service'

printf '%s\n' 'install-test: PASS successful reinstall ordering'

inside_install_dir="$test_dir/inside-install"
inside_event_log="$test_dir/inside-events"
readonly inside_install_dir inside_event_log

prepare_existing "$inside_install_dir" "$inside_event_log"
run_installer_from \
	"$inside_install_dir/AdGuardHome" \
	'..' \
	"$inside_event_log" \
	'0' \
	-r
assert_events 'download
stop
uninstall
unpack
install' "$inside_event_log" \
	'reinstall from the existing directory must preserve the downloaded package'

printf '%s\n' 'install-test: PASS reinstall from existing directory'

failed_install_dir="$test_dir/failed-install"
failed_event_log="$test_dir/failed-events"
readonly failed_install_dir failed_event_log

prepare_existing "$failed_install_dir" "$failed_event_log"

set +e
run_installer "$failed_install_dir" "$failed_event_log" '1' -r
failed_install_status="$?"
set -e
readonly failed_install_status

if [ "$failed_install_status" -eq '0' ]; then
	printf '%s\n' 'install-test: failed download unexpectedly succeeded' 1>&2

	exit 1
fi

if ! [ -L "$failed_install_dir/AdGuardHome/AdGuardHome" ]; then
	printf '%s\n' 'install-test: failed download removed the existing service' 1>&2

	exit 1
fi

assert_events 'download' "$failed_event_log" \
	'failed download must not stop or uninstall the existing service'

printf '%s\n' 'install-test: PASS download failure preserves existing service'

uninstall_dir="$test_dir/uninstall"
uninstall_event_log="$test_dir/uninstall-events"
readonly uninstall_dir uninstall_event_log

prepare_existing "$uninstall_dir" "$uninstall_event_log"
run_installer "$uninstall_dir" "$uninstall_event_log" '0' -u
assert_events 'stop
uninstall' "$uninstall_event_log" 'uninstall-only mode must not download'

printf '%s\n' 'install-test: PASS uninstall-only mode does not download'

existing_dir="$test_dir/existing"
existing_event_log="$test_dir/existing-events"
readonly existing_dir existing_event_log

prepare_existing "$existing_dir" "$existing_event_log"

set +e
run_installer "$existing_dir" "$existing_event_log" '0'
existing_status="$?"
set -e
readonly existing_status

if [ "$existing_status" -eq '0' ]; then
	printf '%s\n' 'install-test: existing installation without -r unexpectedly succeeded' 1>&2

	exit 1
fi

if ! [ -L "$existing_dir/AdGuardHome/AdGuardHome" ]; then
	printf '%s\n' 'install-test: validation failure removed the existing service' 1>&2

	exit 1
fi

assert_events '' "$existing_event_log" \
	'existing installation without -r must fail before download or service changes'

printf '%s\n' 'install-test: PASS existing installation requires -r before download'

fresh_dir="$test_dir/fresh"
fresh_event_log="$test_dir/fresh-events"
readonly fresh_dir fresh_event_log

run_installer "$fresh_dir" "$fresh_event_log" '0'
assert_events 'download
unpack
install' "$fresh_event_log" 'fresh installation behavior changed'

printf '%s\n' 'install-test: PASS fresh installation'
