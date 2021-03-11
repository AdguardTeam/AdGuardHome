package home

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testParseOK(t *testing.T, ss ...string) options {
	t.Helper()

	o, _, err := parse("", ss)
	require.Nil(t, err)

	return o
}

func testParseErr(t *testing.T, descr string, ss ...string) {
	t.Helper()

	_, _, err := parse("", ss)
	require.NotNilf(t, err, "expected an error because %s but no error returned", descr)
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
	assert.Equal(t, "", testParseOK(t).configFilename, "empty is no config filename")
	assert.Equal(t, "path", testParseOK(t, "-c", "path").configFilename, "-c is config filename")
	testParseParamMissing(t, "-c")

	assert.Equal(t, "path", testParseOK(t, "--config", "path").configFilename, "--config is config filename")
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
	assert.Nil(t, testParseOK(t).bindHost, "empty is not host")
	assert.Equal(t, net.IPv4(1, 2, 3, 4), testParseOK(t, "-h", "1.2.3.4").bindHost, "-h is host")
	testParseParamMissing(t, "-h")

	assert.Equal(t, net.IPv4(1, 2, 3, 4), testParseOK(t, "--host", "1.2.3.4").bindHost, "--host is host")
	testParseParamMissing(t, "--host")
}

func TestParseBindPort(t *testing.T) {
	assert.Equal(t, 0, testParseOK(t).bindPort, "empty is port 0")
	assert.Equal(t, 65535, testParseOK(t, "-p", "65535").bindPort, "-p is port")
	testParseParamMissing(t, "-p")

	assert.Equal(t, 65535, testParseOK(t, "--port", "65535").bindPort, "--port is port")
	testParseParamMissing(t, "--port")

	testParseErr(t, "not an int", "-p", "x")
	testParseErr(t, "hex not supported", "-p", "0x100")
	testParseErr(t, "port negative", "-p", "-1")
	testParseErr(t, "port too high", "-p", "65536")
	testParseErr(t, "port too high", "-p", "4294967297")           // 2^32 + 1
	testParseErr(t, "port too high", "-p", "18446744073709551617") // 2^64 + 1
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

func TestParseDisableMemoryOptimization(t *testing.T) {
	assert.False(t, testParseOK(t).disableMemoryOptimization, "empty is not disable update")
	assert.True(t, testParseOK(t, "--no-mem-optimization").disableMemoryOptimization, "--no-mem-optimization is disable update")
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

func TestSerialize(t *testing.T) {
	const reportFmt = "expected %s but got %s"

	testCases := []struct {
		name string
		opts options
		ss   []string
	}{{
		name: "empty",
		opts: options{},
		ss:   []string{},
	}, {
		name: "config_filename",
		opts: options{configFilename: "path"},
		ss:   []string{"-c", "path"},
	}, {
		name: "work_dir",
		opts: options{workDir: "path"},
		ss:   []string{"-w", "path"},
	}, {
		name: "bind_host",
		opts: options{bindHost: net.IP{1, 2, 3, 4}},
		ss:   []string{"-h", "1.2.3.4"},
	}, {
		name: "bind_port",
		opts: options{bindPort: 666},
		ss:   []string{"-p", "666"},
	}, {
		name: "log_file",
		opts: options{logFile: "path"},
		ss:   []string{"-l", "path"},
	}, {
		name: "pid_file",
		opts: options{pidFile: "path"},
		ss:   []string{"--pidfile", "path"},
	}, {
		name: "disable_update",
		opts: options{disableUpdate: true},
		ss:   []string{"--no-check-update"},
	}, {
		name: "control_action",
		opts: options{serviceControlAction: "run"},
		ss:   []string{"-s", "run"},
	}, {
		name: "glinet_mode",
		opts: options{glinetMode: true},
		ss:   []string{"--glinet"},
	}, {
		name: "disable_mem_opt",
		opts: options{disableMemoryOptimization: true},
		ss:   []string{"--no-mem-optimization"},
	}, {
		name: "multiple",
		opts: options{
			serviceControlAction:      "run",
			configFilename:            "config",
			workDir:                   "work",
			pidFile:                   "pid",
			disableUpdate:             true,
			disableMemoryOptimization: true,
		},
		ss: []string{
			"-c", "config",
			"-w", "work",
			"-s", "run",
			"--pidfile", "pid",
			"--no-check-update",
			"--no-mem-optimization",
		},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := serialize(tc.opts)
			require.Lenf(t, result, len(tc.ss), reportFmt, tc.ss, result)

			for i, r := range result {
				assert.Equalf(t, tc.ss[i], r, reportFmt, tc.ss, result)
			}
		})
	}
}
