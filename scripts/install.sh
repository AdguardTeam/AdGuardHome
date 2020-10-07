#!/bin/sh

# AdGuardHome installation script
#
# 1. Download the package
# 2. Unpack it
# 3. Install as a service
#
# Requirements:
# . bash
# . which
# . printf
# . uname
# . id
# . head, tail
# . curl
# . tar or unzip
# . rm

set -e

log_info()
{
    printf "[info] %s\\n" "$1"
}

log_error()
{
    printf "[error] %s\\n" "$1"
}

# Get OS
# Return: darwin, linux, freebsd
detect_os()
{
	UNAME_S="$(uname -s)"
	OS=
	case "$UNAME_S" in
		Linux)
			OS=linux
			;;

		FreeBSD)
			OS=freebsd
			;;

		Darwin)
			OS=darwin
			;;

		*)
			return 1
			;;

	esac

	echo $OS
}

# Get CPU endianness
# Return: le, ""
cpu_little_endian()
{
	ENDIAN_FLAG="$(head -c 6 /bin/bash | tail -c 1)"
	if [ "$ENDIAN_FLAG" = "$(printf '\001')" ]; then
		echo 'le'
		return 0
	fi
}

# Get CPU
# Return: amd64, 386, armv5, armv6, armv7, arm64, mips_softfloat, mipsle_softfloat, mips64_softfloat, mips64le_softfloat
detect_cpu()
{
	UNAME_M="$(uname -m)"
	CPU=

	case "$UNAME_M" in

		x86_64 | x86-64 | x64 | amd64)
			CPU=amd64
			;;

		i386 | i486 | i686 | i786 | x86)
			CPU=386
			;;

		armv5l)
			CPU=armv5
			;;

		armv6l)
			CPU=armv6
			;;

		armv7l | armv8l)
			CPU=armv7
			;;

		aarch64)
			CPU=arm64
			;;

		mips)
			LE=$(cpu_little_endian)
			CPU=mips${LE}_softfloat
			;;

		mips64)
			LE=$(cpu_little_endian)
			CPU=mips64${LE}_softfloat
			;;

		*)
			return 1

	esac

	echo "${CPU}"
}

# Get package file name extension
# Return: tar.gz, zip
package_extension()
{
	if [ "$OS" = "darwin" ]; then
		echo "zip"
		return 0
	fi
	echo "tar.gz"
}

# Download data to a file
# Use: download URL OUTPUT
download()
{
	log_info "Downloading package from $1 -> $2"
	if is_command curl ; then
		curl -s "$1" --output "$2" || error_exit "Failed to download $1"
	else
		error_exit "curl is necessary to install AdGuard Home"
	fi
}

# Unpack package to a directory
# Use: unpack INPUT OUTPUT_DIR PKG_EXT
unpack()
{
	log_info "Unpacking package from $1 -> $2"
	mkdir -p "$2"
	if [ "$3" = "zip" ]; then
		unzip -qq "$1" -d "$2" || return 1
	elif [ "$3" = "tar.gz" ]; then
		tar xzf "$1" -C "$2" || return 1
	else
		return 1
	fi
}

# Print error message and exit
# Use: error_exit MESSAGE
error_exit()
{
	log_error "$1"
	exit 1
}

# Check if command exists
# Use: is_command COMMAND
is_command() {
    check_command="$1"
    command -v "${check_command}" >/dev/null 2>&1
}

# Entry point
main() {
    log_info "Starting AdGuard Home installation script"

    CHANNEL=${1}
    if [ "${CHANNEL}" != "beta" ] && [ "${CHANNEL}" != "edge" ]; then
        CHANNEL=release
    fi
    log_info "Channel ${CHANNEL}"

    OS=$(detect_os) || error_exit "Cannot detect your OS"
    CPU=$(detect_cpu) || error_exit "Cannot detect your CPU"
    PKG_EXT=$(package_extension)
    PKG_NAME=AdGuardHome_${OS}_${CPU}.${PKG_EXT}

    SCRIPT_URL="https://raw.githubusercontent.com/AdguardTeam/AdGuardHome/master/scripts/install.sh"
    URL="https://static.adguard.com/adguardhome/${CHANNEL}/${PKG_NAME}"
    OUT_DIR=/opt

    # Root check
    if [ "$(id -u)" -eq 0 ]; then
        log_info "Script called with root privileges"
    else
        if is_command sudo ; then
            log_info "Please note, that AdGuard Home requires root privileges to install using this script."
            log_info "Restarting with root privileges"

            exec curl -sSL ${SCRIPT_URL} | sudo sh -s "$@"
            exit $?
        else
            log_info "Root privileges are required to install AdGuard Home using this installer."
            log_info "Please, re-run this script as root."
            exit 1
        fi
    fi

    log_info "AdGuard Home will be installed to ${OUT_DIR}/AdGuardHome"

    [ -d "${OUT_DIR}/AdGuardHome" ] && error_exit "Directory ${OUT_DIR}/AdGuardHome already exists, abort installation"

    download "${URL}" "${PKG_NAME}" || error_exit "Cannot download the package"

    unpack "${PKG_NAME}" "${OUT_DIR}" "${PKG_EXT}" || error_exit "Cannot unpack the package"

    # Install AdGuard Home service and run it
    ${OUT_DIR}/AdGuardHome/AdGuardHome -s install || error_exit "Cannot install AdGuardHome as a service"

    rm "${PKG_NAME}"

    log_info "AdGuard Home is now installed and running."
    log_info "You can control the service status with the following commands:"
    log_info "  sudo ${OUT_DIR}/AdGuardHome/AdGuardHome -s start|stop|restart|status|install|uninstall"
}

main "$@"