#!/bin/sh

# AdGuard Home Installation Script

# Exit the script if a pipeline fails (-e), prevent accidental filename
# expansion (-f), and consider undefined variables as errors (-u).
set -e -f -u

# Function log is an echo wrapper that writes to stderr if the caller
# requested verbosity level greater than 0.  Otherwise, it does nothing.
log() {
	if [ "$verbose" -gt '0' ]
	then
		echo "$1" 1>&2
	fi
}

# Function error_exit is an echo wrapper that writes to stderr and stops the
# script execution with code 1.
error_exit() {
	echo "$1" 1>&2

	exit 1
}

# Function usage prints the note about how to use the script.
#
# TODO(e.burkov): Document each option.
usage() {
	echo 'install.sh: usage: [-c channel] [-C cpu_type] [-h] [-O os] [-o output_dir]'\
		'[-r|-R] [-u|-U] [-v|-V]' 1>&2

	exit 2
}

# Function is_command checks if the command exists on the machine.
is_command() {
	command -v "$1" >/dev/null 2>&1
}

# Function is_little_endian checks if the CPU is little-endian.
is_little_endian() {
	[ "$( head -c 6 /bin/sh | tail -c 1 )" = "$( printf '\001' )" ]
}

# Function check_required checks if the required software is available on the
# machine.  The required software:
#
#   curl
#   unzip (macOS) / tar (other unices)
#
check_required() {
	required_darwin="unzip"
	required_unix="tar"
	readonly required_darwin required_unix

	# Split with space.
	required="curl"
	case "$os"
	in
	('freebsd'|'linux'|'openbsd')
		required="$required $required_unix"
		;;
	('darwin')
		required="$required $required_darwin"
		;;
	(*)
		# Generally shouldn't happen, since the OS has already been
		# validated.
		error_exit "unsupported operating system: '$os'"
		;;
	esac

	# Don't use quotes to get word splitting.
	for cmd in ${required}
	do
		log "checking $cmd"
		if ! is_command "$cmd"
		then
			log "the full list of required software: [$required]"

			error_exit "$cmd is required to install AdGuard Home via this script"
		fi
	done
}

# Function check_out_dir requires the output directory to be set and exist.
check_out_dir() {
	if [ "$out_dir" = '' ]
	then
		error_exit 'output directory should be presented'
	fi

	if ! [ -d "$out_dir" ]
	then
		log "$out_dir directory will be created"
	fi
}

# Function parse_opts parses the options list and validates it's combinations.
parse_opts() {
	while getopts "C:c:hO:o:rRuUvV" opt "$@"
	do
		case "$opt"
		in
		(C)
			cpu="$OPTARG"
			;;
		(c)
			channel="$OPTARG"
			;;
		(h)
			usage
			;;
		(O)
			os="$OPTARG"
			;;
		(o)
			out_dir="$OPTARG"
			;;
		(R)
			reinstall='0'
			;;
		(U)
			uninstall='0'
			;;
		(r)
			reinstall='1'
			;;
		(u)
			uninstall='1'
			;;
		(V)
			verbose='0'
			;;
		(v)
			verbose='1'
			;;
		(*)
			log "bad option $OPTARG"

			usage
			;;
		esac
	done

	if [ "$uninstall" -eq '1' ] && [ "$reinstall" -eq '1' ]
	then
		error_exit 'the -r and -u options are mutually exclusive'
	fi
}

# Function set_channel sets the channel if needed and validates the value.
set_channel() {
	# Validate.
	case "$channel"
	in
	('development'|'edge'|'beta'|'release')
		# All is well, go on.
		;;
	(*)
		error_exit \
"invalid channel '$channel'
supported values are 'development', 'edge', 'beta', and 'release'"
		;;
	esac

	# Log.
	log "channel: $channel"
}

# Function set_os sets the os if needed and validates the value.
set_os() {
	# Set if needed.
	if [ "$os" = '' ]
	then
		os="$( uname -s )"
		case "$os"
		in
		('Darwin')
			os='darwin'
			;;
		('FreeBSD')
			os='freebsd'
			;;
		('Linux')
			os='linux'
			;;
		('OpenBSD')
			os='openbsd'
			;;
		esac
	fi

	# Validate.
	case "$os"
	in
	('darwin'|'freebsd'|'linux'|'openbsd')
		# All right, go on.
		;;
	(*)
		error_exit "unsupported operating system: '$os'"
		;;
	esac

	# Log.
	log "operating system: $os"
}

# Function set_cpu sets the cpu if needed and validates the value.
set_cpu() {
	# Set if needed.
	if [ "$cpu" = '' ]
	then
		cpu="$( uname -m )"
		case "$cpu"
		in
		('x86_64'|'x86-64'|'x64'|'amd64')
			cpu='amd64'
			;;
		('i386'|'i486'|'i686'|'i786'|'x86')
			cpu='386'
			;;
		('armv5l')
			cpu='armv5'
			;;
		('armv6l')
			cpu='armv6'
			;;
		('armv7l' | 'armv8l')
			cpu='armv7'
			;;
		('aarch64'|'arm64')
			cpu='arm64'
			;;
		('mips'|'mips64')
			if is_little_endian
			then
				cpu="${cpu}le"
			fi
			cpu="${cpu}_softfloat"
			;;
		esac
	fi

	# Validate.
	case "$cpu"
	in
	('amd64'|'386'|'armv5'|'armv6'|'armv7'|'arm64')
		# All right, go on.
		;;
	('mips64le_softfloat'|'mips64_softfloat'|'mipsle_softfloat'|'mips_softfloat')
		# That's right too.
		;;
	(*)
		error_exit "unsupported cpu type: $cpu"
		;;
	esac

	# Log.
	log "cpu type: $cpu"
}

# Function fix_darwin performs some configuration changes for macOS if needed.
#
# TODO(a.garipov): Remove after the final v0.107.0 release.
#
# See https://github.com/AdguardTeam/AdGuardHome/issues/2443.
fix_darwin() {
	if [ "$os" != 'darwin' ]
	then
		return 0
	fi

	if [ "$cpu" = 'arm64' ]
	then
		case "$channel"
		in
		('beta'|'development'|'edge')
			# Everything is fine, we have Apple Silicon support on
			# these channels.
			;;
		('release')
			cpu='amd64'
			log "use $cpu build on Mac M1 until the native ARM support is added."
			;;
		(*)
			# Generally shouldn't happen, since the release channel
			# has already been validated.
			error_exit "invalid channel '$channel'"
			;;
		esac
	fi

	# Set the package extension.
	pkg_ext='zip'

	# It is important to install AdGuard Home into the /Applications
	# directory on macOS.  Otherwise, it may not grant enough privileges to
	# the AdGuard Home.
	out_dir='/Applications'
}

# Function fix_freebsd performs some fixes to make it work on FreeBSD.
fix_freebsd() {
	if ! [ "$os" = 'freebsd' ]
	then
		return 0
	fi

	rcd='/usr/local/etc/rc.d'
	readonly rcd

	if ! [ -d "$rcd" ]
	then
		mkdir "$rcd"
	fi
}

# Function set_sudo_cmd sets the appropriate command to run a command under
# superuser privileges.
set_sudo_cmd() {
	case "$os"
	in
	('openbsd')
		sudo_cmd='doas'
		;;
	('darwin'|'freebsd'|'linux')
		# Go on and use the default, sudo.
		;;
	(*)
		error_exit "unsupported operating system: '$os'"
		;;
	esac
}

# Function configure sets the script's configuration.
configure() {
	set_channel
	set_os
	set_cpu
	fix_darwin
	set_sudo_cmd
	check_out_dir

	pkg_name="AdGuardHome_${os}_${cpu}.${pkg_ext}"
	url="https://static.adguard.com/adguardhome/${channel}/${pkg_name}"
	agh_dir="${out_dir}/AdGuardHome"
	readonly pkg_name url agh_dir

	log "AdGuard Home will be installed into $agh_dir"
}

# Function is_root checks for root privileges to be granted.
is_root() {
	if [ "$( id -u )" -eq '0' ]
	then
		log 'script is executed with root privileges'

		return 0
	fi

	if is_command "$sudo_cmd"
	then
		log 'note that AdGuard Home requires root privileges to install using this script'

		return 1
	fi

	error_exit \
'root privileges are required to install AdGuard Home using this script
please, restart it with root privileges'
}

# Function rerun_with_root downloads the script, runs it with root privileges,
# and exits the current script.  It passes the necessary configuration of the
# current script to the child script.
#
# TODO(e.burkov): Try to avoid restarting.
rerun_with_root() {
	script_url=\
'https://raw.githubusercontent.com/AdguardTeam/AdGuardHome/master/scripts/install.sh'
	readonly script_url

	r='-R'
	if [ "$reinstall" -eq '1' ]
	then
		r='-r'
	fi

	u='-U'
	if [ "$uninstall" -eq '1' ]
	then
		u='-u'
	fi

	v='-V'
	if [ "$verbose" -eq '1' ]
	then
		v='-v'
	fi

	readonly r u v

	log 'restarting with root privileges'
	
	# Group curl together with an echo, so that if curl fails before
	# producing any output, the echo prints an exit command for the
	# following shell to execute to prevent it from getting an empty input
	# and exiting with a zero code in that case.
	{ curl -L -S -s "$script_url" || echo 'exit 1'; }\
		| $sudo_cmd sh -s -- -c "$channel" -C "$cpu" -O "$os" -o "$out_dir" "$r" "$u" "$v"

	# Exit the script.  Since if the code of the previous pipeline is
	# non-zero, the execution won't reach this point thanks to set -e, exit
	# with zero.
	exit 0
}

# Function download downloads the file from the URL and saves it to the
# specified filepath.
download() {
	log "downloading package from $url -> $pkg_name"

	if ! curl -s "$url" --output "$pkg_name"
	then
		error_exit "cannot download the package from $url into $pkg_name"
	fi
}

# Function unpack unpacks the passed archive depending on it's extension.
unpack() {
	log "unpacking package from $pkg_name into $out_dir"
	if ! mkdir -p "$out_dir"
	then
		error_exit "cannot create directory at the $out_dir"
	fi

	case "$pkg_ext"
	in
	('zip')
		unzip "$pkg_name" -d "$out_dir"
		;;
	('tar.gz')
		tar -C "$out_dir" -f "$pkg_name" -x -z
		;;
	(*)
		error_exit "unexpected package extension: '$pkg_ext'"
		;;
	esac

	rm "$pkg_name"
}

# Function handle_existing detects the existing AGH installation and takes care
# of removing it if needed.
handle_existing() {
	if ! [ -d "$agh_dir" ]
	then
		log 'no need to uninstall'

		if [ "$uninstall" -eq '1' ]
		then
			exit 0
		fi

		return 0
	fi

	if [ "$( ls -1 -A $agh_dir )" != '' ]
	then
		log 'the existing AdGuard Home installation is detected'

		if [ "$reinstall" -ne '1' ] && [ "$uninstall" -ne '1' ]
		then
			error_exit \
"to reinstall/uninstall the AdGuard Home using this script specify one of the '-r' or '-u' flags"
		fi

		if ( cd "$agh_dir" && ! ./AdGuardHome -s stop || ! ./AdGuardHome -s uninstall )
		then
			# It doesn't terminate the script since it is possible
			# that AGH just not installed as service but appearing
			# in the directory.
			log "cannot uninstall AdGuard Home from $agh_dir"
		fi

		rm -r "$agh_dir"

		log 'AdGuard Home was successfully uninstalled'
	fi

	if [ "$uninstall" -eq '1' ]
	then
		exit 0
	fi
}

# Function install_service tries to install AGH as service.
install_service() {
	# Installing as root is required at least on FreeBSD.
	#
	# TODO(e.burkov): Think about AGH's output suppressing with no verbose
	# flag.
	if ( cd "$agh_dir" && $sudo_cmd ./AdGuardHome -s install )
	then
		return 0
	fi

	rm -r "$agh_dir"

	# Some devices detected to have armv7 CPU face the compatibility
	# issues with actual armv7 builds.  We should try to install the
	# armv5 binary instead.
	#
	# See https://github.com/AdguardTeam/AdGuardHome/issues/2542.
	if [ "$cpu" = 'armv7' ]
	then
		cpu='armv5'
		reinstall='1'

		log "trying to use $cpu cpu"

		rerun_with_root
	fi

	error_exit 'cannot install AdGuardHome as a service'
}



# Entrypoint

# Set default values of configuration variables.
channel='release'
reinstall='0'
uninstall='0'
verbose='0'
cpu=''
os=''
out_dir='/opt'
pkg_ext='tar.gz'
sudo_cmd='sudo'

parse_opts "$@"

echo 'starting AdGuard Home installation script'

configure
check_required

if ! is_root
then
	rerun_with_root
fi
# Needs rights.
fix_freebsd

handle_existing

download
unpack

install_service

echo "\
AdGuard Home is now installed and running
you can control the service status with the following commands:
$sudo_cmd ${agh_dir}/AdGuardHome -s start|stop|restart|status|install|uninstall"
