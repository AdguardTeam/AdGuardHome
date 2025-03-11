package home

import (
	"fmt"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testParseOK(t *testing.T, ss ...string) options {
	t.Helper()

	o, _, err := parseCmdOpts("", ss)
	require.NoError(t, err)

	return o
}

func testParseErr(t *testing.T, descr string, ss ...string) {
	t.Helper()

	_, _, err := parseCmdOpts("", ss)
	require.Error(t, err)
}

func testParseParamMissing(t *testing.T, param string) {
	t.Helper()

	testParseErr(t, fmt.Sprintf("%s parameter missing", param), param)
}

func TestParseVerbose(t *testing.T) {
	assert.False(t, testParseOK(t).verbose, "empty is not verbose")
	assert.True(t, testParseOK(t, "-v").verbose, "-v is verbose")
	assert.True(t, testParseOK(t, "--verbose").verbose, "--verbose is verbose")
}

func TestParseConfigFilename(t *testing.T) {
	assert.Equal(t, "", testParseOK(t).confFilename, "empty is no config filename")
	assert.Equal(t, "path", testParseOK(t, "-c", "path").confFilename, "-c is config filename")
	testParseParamMissing(t, "-c")

	assert.Equal(t, "path", testParseOK(t, "--config", "path").confFilename, "--config is config filename")
	testParseParamMissing(t, "--config")
}

func TestParseWorkDir(t *testing.T) {
	assert.Equal(t, "", testParseOK(t).workDir, "empty is no work dir")
	assert.Equal(t, "path", testParseOK(t, "-w", "path").workDir, "-w is work dir")
	testParseParamMissing(t, "-w")

	assert.Equal(t, "path", testParseOK(t, "--work-dir", "path").workDir, "--work-dir is work dir")
	testParseParamMissing(t, "--work-dir")
}

func TestParseBindHost(t *testing.T) {
	wantAddr := netip.MustParseAddr("1.2.3.4")

	assert.Zero(t, testParseOK(t).bindHost, "empty is not host")
	assert.Equal(t, wantAddr, testParseOK(t, "-h", "1.2.3.4").bindHost, "-h is host")
	testParseParamMissing(t, "-h")

	assert.Equal(t, wantAddr, testParseOK(t, "--host", "1.2.3.4").bindHost, "--host is host")
	testParseParamMissing(t, "--host")
}

func TestParseBindPort(t *testing.T) {
	assert.Equal(t, uint16(0), testParseOK(t).bindPort, "empty is port 0")
	assert.Equal(t, uint16(65535), testParseOK(t, "-p", "65535").bindPort, "-p is port")
	testParseParamMissing(t, "-p")

	assert.Equal(t, uint16(65535), testParseOK(t, "--port", "65535").bindPort, "--port is port")
	testParseParamMissing(t, "--port")

	testParseErr(t, "not an int", "-p", "x")
	testParseErr(t, "hex not supported", "-p", "0x100")
	testParseErr(t, "port negative", "-p", "-1")
	testParseErr(t, "port too high", "-p", "65536")
	testParseErr(t, "port too high", "-p", "4294967297")           // 2^32 + 1
	testParseErr(t, "port too high", "-p", "18446744073709551617") // 2^64 + 1
}

func TestParseBindAddr(t *testing.T) {
	wantAddrPort := netip.MustParseAddrPort("1.2.3.4:8089")

	assert.Zero(t, testParseOK(t).bindAddr, "empty is not web-addr")

	assert.Equal(t, wantAddrPort, testParseOK(t, "--web-addr", "1.2.3.4:8089").bindAddr)
	assert.Equal(t, netip.MustParseAddrPort("1.2.3.4:0"), testParseOK(t, "--web-addr", "1.2.3.4:0").bindAddr)
	testParseParamMissing(t, "-web-addr")

	testParseErr(t, "not an int", "--web-addr", "1.2.3.4:x")
	testParseErr(t, "hex not supported", "--web-addr", "1.2.3.4:0x100")
	testParseErr(t, "port negative", "--web-addr", "1.2.3.4:-1")
	testParseErr(t, "port too high", "--web-addr", "1.2.3.4:65536")
	testParseErr(t, "port too high", "--web-addr", "1.2.3.4:4294967297")           // 2^32 + 1
	testParseErr(t, "port too high", "--web-addr", "1.2.3.4:18446744073709551617") // 2^64 + 1
}

func TestParseLogfile(t *testing.T) {
	assert.Equal(t, "", testParseOK(t).logFile, "empty is no log file")
	assert.Equal(t, "path", testParseOK(t, "-l", "path").logFile, "-l is log file")
	assert.Equal(t, "path", testParseOK(t, "--logfile", "path").logFile, "--logfile is log file")
}

func TestParsePidfile(t *testing.T) {
	assert.Equal(t, "", testParseOK(t).pidFile, "empty is no pid file")
	assert.Equal(t, "path", testParseOK(t, "--pidfile", "path").pidFile, "--pidfile is pid file")
}

func TestParseCheckConfig(t *testing.T) {
	assert.False(t, testParseOK(t).checkConfig, "empty is not check config")
	assert.True(t, testParseOK(t, "--check-config").checkConfig, "--check-config is check config")
}

func TestParseDisableUpdate(t *testing.T) {
	assert.False(t, testParseOK(t).disableUpdate, "empty is not disable update")
	assert.True(t, testParseOK(t, "--no-check-update").disableUpdate, "--no-check-update is disable update")
}

func TestParsePerformUpdate(t *testing.T) {
	assert.False(t, testParseOK(t).performUpdate, "empty is not perform update")
	assert.True(t, testParseOK(t, "--update").performUpdate, "--update is perform update")
}

// TODO(e.burkov):  Remove after v0.108.0.
func TestParseDisableMemoryOptimization(t *testing.T) {
	o, eff, err := parseCmdOpts("", []string{"--no-mem-optimization"})
	require.NoError(t, err)

	assert.Nil(t, eff)
	assert.Zero(t, o)
}

func TestParseService(t *testing.T) {
	assert.Equal(t, "", testParseOK(t).serviceControlAction, "empty is not service cmd")
	assert.Equal(t, "cmd", testParseOK(t, "-s", "cmd").serviceControlAction, "-s is service cmd")
	assert.Equal(t, "cmd", testParseOK(t, "--service", "cmd").serviceControlAction, "--service is service cmd")
}

func TestParseGLInet(t *testing.T) {
	assert.False(t, testParseOK(t).glinetMode, "empty is not GL-Inet mode")
	assert.True(t, testParseOK(t, "--glinet").glinetMode, "--glinet is GL-Inet mode")
}

func TestParseUnknown(t *testing.T) {
	testParseErr(t, "unknown word", "x")
	testParseErr(t, "unknown short", "-x")
	testParseErr(t, "unknown long", "--x")
	testParseErr(t, "unknown triple", "---x")
	testParseErr(t, "unknown plus", "+x")
	testParseErr(t, "unknown dash", "-")
}

func TestOptsToArgs(t *testing.T) {
	testCases := []struct {
		name string
		args []string
		opts options
	}{{
		name: "empty",
		args: []string{},
		opts: options{},
	}, {
		name: "config_filename",
		args: []string{"-c", "path"},
		opts: options{confFilename: "path"},
	}, {
		name: "work_dir",
		args: []string{"-w", "path"},
		opts: options{workDir: "path"},
	}, {
		name: "bind_host",
		opts: options{bindHost: netip.MustParseAddr("1.2.3.4")},
		args: []string{"-h", "1.2.3.4"},
	}, {
		name: "bind_port",
		args: []string{"-p", "666"},
		opts: options{bindPort: 666},
	}, {
		name: "web-addr",
		args: []string{"--web-addr", "1.2.3.4:8080"},
		opts: options{bindAddr: netip.MustParseAddrPort("1.2.3.4:8080")},
	}, {
		name: "log_file",
		args: []string{"-l", "path"},
		opts: options{logFile: "path"},
	}, {
		name: "pid_file",
		args: []string{"--pidfile", "path"},
		opts: options{pidFile: "path"},
	}, {
		name: "disable_update",
		args: []string{"--no-check-update"},
		opts: options{disableUpdate: true},
	}, {
		name: "perform_update",
		args: []string{"--update"},
		opts: options{performUpdate: true},
	}, {
		name: "control_action",
		args: []string{"-s", "run"},
		opts: options{serviceControlAction: "run"},
	}, {
		name: "glinet_mode",
		args: []string{"--glinet"},
		opts: options{glinetMode: true},
	}, {
		name: "multiple",
		args: []string{
			"-c", "config",
			"-w", "work",
			"-s", "run",
			"--pidfile", "pid",
			"--no-check-update",
		},
		opts: options{
			serviceControlAction: "run",
			confFilename:         "config",
			workDir:              "work",
			pidFile:              "pid",
			disableUpdate:        true,
		},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := optsToArgs(tc.opts)
			assert.ElementsMatch(t, tc.args, result)
		})
	}
}
