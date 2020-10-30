package home

import (
	"fmt"
	"testing"
)

func testParseOk(t *testing.T, ss ...string) options {
	o, _, err := parse("", ss)
	if err != nil {
		t.Fatal(err.Error())
	}
	return o
}

func testParseErr(t *testing.T, descr string, ss ...string) {
	_, _, err := parse("", ss)
	if err == nil {
		t.Fatalf("expected an error because %s but no error returned", descr)
	}
}

func testParseParamMissing(t *testing.T, param string) {
	testParseErr(t, fmt.Sprintf("%s parameter missing", param), param)
}

func TestParseVerbose(t *testing.T) {
	if testParseOk(t).verbose {
		t.Fatal("empty is not verbose")
	}
	if !testParseOk(t, "-v").verbose {
		t.Fatal("-v is verbose")
	}
	if !testParseOk(t, "--verbose").verbose {
		t.Fatal("--verbose is verbose")
	}
}

func TestParseConfigFilename(t *testing.T) {
	if testParseOk(t).configFilename != "" {
		t.Fatal("empty is no config filename")
	}
	if testParseOk(t, "-c", "path").configFilename != "path" {
		t.Fatal("-c is config filename")
	}
	testParseParamMissing(t, "-c")
	if testParseOk(t, "--config", "path").configFilename != "path" {
		t.Fatal("--configFilename is config filename")
	}
	testParseParamMissing(t, "--config")
}

func TestParseWorkDir(t *testing.T) {
	if testParseOk(t).workDir != "" {
		t.Fatal("empty is no work dir")
	}
	if testParseOk(t, "-w", "path").workDir != "path" {
		t.Fatal("-w is work dir")
	}
	testParseParamMissing(t, "-w")
	if testParseOk(t, "--work-dir", "path").workDir != "path" {
		t.Fatal("--work-dir is work dir")
	}
	testParseParamMissing(t, "--work-dir")
}

func TestParseBindHost(t *testing.T) {
	if testParseOk(t).bindHost != "" {
		t.Fatal("empty is no host")
	}
	if testParseOk(t, "-h", "addr").bindHost != "addr" {
		t.Fatal("-h is host")
	}
	testParseParamMissing(t, "-h")
	if testParseOk(t, "--host", "addr").bindHost != "addr" {
		t.Fatal("--host is host")
	}
	testParseParamMissing(t, "--host")
}

func TestParseBindPort(t *testing.T) {
	if testParseOk(t).bindPort != 0 {
		t.Fatal("empty is port 0")
	}
	if testParseOk(t, "-p", "65535").bindPort != 65535 {
		t.Fatal("-p is port")
	}
	testParseParamMissing(t, "-p")
	if testParseOk(t, "--port", "65535").bindPort != 65535 {
		t.Fatal("--port is port")
	}
	testParseParamMissing(t, "--port")
}

func TestParseBindPortBad(t *testing.T) {
	testParseErr(t, "not an int", "-p", "x")
	testParseErr(t, "hex not supported", "-p", "0x100")
	testParseErr(t, "port negative", "-p", "-1")
	testParseErr(t, "port too high", "-p", "65536")
	testParseErr(t, "port too high", "-p", "4294967297")           // 2^32 + 1
	testParseErr(t, "port too high", "-p", "18446744073709551617") // 2^64 + 1
}

func TestParseLogfile(t *testing.T) {
	if testParseOk(t).logFile != "" {
		t.Fatal("empty is no log file")
	}
	if testParseOk(t, "-l", "path").logFile != "path" {
		t.Fatal("-l is log file")
	}
	if testParseOk(t, "--logfile", "path").logFile != "path" {
		t.Fatal("--logfile is log file")
	}
}

func TestParsePidfile(t *testing.T) {
	if testParseOk(t).pidFile != "" {
		t.Fatal("empty is no pid file")
	}
	if testParseOk(t, "--pidfile", "path").pidFile != "path" {
		t.Fatal("--pidfile is pid file")
	}
}

func TestParseCheckConfig(t *testing.T) {
	if testParseOk(t).checkConfig {
		t.Fatal("empty is not check config")
	}
	if !testParseOk(t, "--check-config").checkConfig {
		t.Fatal("--check-config is check config")
	}
}

func TestParseDisableUpdate(t *testing.T) {
	if testParseOk(t).disableUpdate {
		t.Fatal("empty is not disable update")
	}
	if !testParseOk(t, "--no-check-update").disableUpdate {
		t.Fatal("--no-check-update is disable update")
	}
}

func TestParseDisableMemoryOptimization(t *testing.T) {
	if testParseOk(t).disableMemoryOptimization {
		t.Fatal("empty is not disable update")
	}
	if !testParseOk(t, "--no-mem-optimization").disableMemoryOptimization {
		t.Fatal("--no-mem-optimization is disable update")
	}
}

func TestParseService(t *testing.T) {
	if testParseOk(t).serviceControlAction != "" {
		t.Fatal("empty is no service command")
	}
	if testParseOk(t, "-s", "command").serviceControlAction != "command" {
		t.Fatal("-s is service command")
	}
	if testParseOk(t, "--service", "command").serviceControlAction != "command" {
		t.Fatal("--service is service command")
	}
}

func TestParseGLInet(t *testing.T) {
	if testParseOk(t).glinetMode {
		t.Fatal("empty is not GL-Inet mode")
	}
	if !testParseOk(t, "--glinet").glinetMode {
		t.Fatal("--glinet is GL-Inet mode")
	}
}

func TestParseUnknown(t *testing.T) {
	testParseErr(t, "unknown word", "x")
	testParseErr(t, "unknown short", "-x")
	testParseErr(t, "unknown long", "--x")
	testParseErr(t, "unknown triple", "---x")
	testParseErr(t, "unknown plus", "+x")
	testParseErr(t, "unknown dash", "-")
}

func testSerialize(t *testing.T, o options, ss ...string) {
	result := serialize(o)
	if len(result) != len(ss) {
		t.Fatalf("expected %s but got %s", ss, result)
	}
	for i, r := range result {
		if r != ss[i] {
			t.Fatalf("expected %s but got %s", ss, result)
		}
	}
}

func TestSerializeEmpty(t *testing.T) {
	testSerialize(t, options{})
}

func TestSerializeConfigFilename(t *testing.T) {
	testSerialize(t, options{configFilename: "path"}, "-c", "path")
}

func TestSerializeWorkDir(t *testing.T) {
	testSerialize(t, options{workDir: "path"}, "-w", "path")
}

func TestSerializeBindHost(t *testing.T) {
	testSerialize(t, options{bindHost: "addr"}, "-h", "addr")
}

func TestSerializeBindPort(t *testing.T) {
	testSerialize(t, options{bindPort: 666}, "-p", "666")
}

func TestSerializeLogfile(t *testing.T) {
	testSerialize(t, options{logFile: "path"}, "-l", "path")
}

func TestSerializePidfile(t *testing.T) {
	testSerialize(t, options{pidFile: "path"}, "--pidfile", "path")
}

func TestSerializeCheckConfig(t *testing.T) {
	testSerialize(t, options{checkConfig: true}, "--check-config")
}

func TestSerializeDisableUpdate(t *testing.T) {
	testSerialize(t, options{disableUpdate: true}, "--no-check-update")
}

func TestSerializeService(t *testing.T) {
	testSerialize(t, options{serviceControlAction: "run"}, "-s", "run")
}

func TestSerializeGLInet(t *testing.T) {
	testSerialize(t, options{glinetMode: true}, "--glinet")
}

func TestSerializeDisableMemoryOptimization(t *testing.T) {
	testSerialize(t, options{disableMemoryOptimization: true}, "--no-mem-optimization")
}

func TestSerializeMultiple(t *testing.T) {
	testSerialize(t, options{
		serviceControlAction:      "run",
		configFilename:            "config",
		workDir:                   "work",
		pidFile:                   "pid",
		disableUpdate:             true,
		disableMemoryOptimization: true,
	}, "-c", "config", "-w", "work", "-s", "run", "--pidfile", "pid", "--no-check-update", "--no-mem-optimization")
}
