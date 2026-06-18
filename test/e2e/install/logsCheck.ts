import assert from 'node:assert/strict';

export interface CommandResult {
  stdout: string;
  stderr: string;
}

export type CommandRunner = (command: string) => Promise<CommandResult>;
export type FileReader = (path: string) => Promise<string>;
export type FileWriter = (path: string, content: string) => Promise<void>;

export interface LogsCheckScenario {
  serviceName?: string;
  binaryPath?: string;
  configPath?: string;
}

export interface LogsCheckContext {
  runCommand: CommandRunner;
  readFile: FileReader;
  writeFile: FileWriter;
}

export interface LogsCheckResult {
  initialLogs: string;
  updatedLogs: string;
  updatedConfig: string;
}

/**
 * Enables verbose logging in AdGuardHome.yaml payload.
 * Keeps original formatting and updates only the first matched "verbose" key.
 */
export function enableVerboseLogging(configYaml: string): string {
  const verbosePattern = /^(\s*verbose:\s*)(true|false)(\s*(?:#.*)?)$/m;

  const match = configYaml.match(verbosePattern);
  assert.ok(match, 'AdGuardHome.yaml must contain a top-level "verbose" setting');

  const currentValue = match?.[2];
  if (currentValue === 'true') {
    return configYaml;
  }

  return configYaml.replace(verbosePattern, '$1true$3');
}

/**
 * Runs end-to-end shell flow for testcase 4033.
 *
 * Flow:
 * 1) Read current logs via journalctl.
 * 2) Stop AdGuard Home.
 * 3) Enable verbose logging in config.
 * 4) Start AdGuard Home.
 * 5) Read logs again via journalctl.
 */
export async function runLogsCheckTestCase(
  context: LogsCheckContext,
  scenario: LogsCheckScenario = {},
): Promise<LogsCheckResult> {
  const serviceName = scenario.serviceName ?? 'AdGuardHome';
  const binaryPath = scenario.binaryPath ?? './AdGuardHome';
  const configPath = scenario.configPath ?? 'AdGuardHome.yaml';

  const initialLogsResult = await context.runCommand(`journalctl -u ${serviceName}`);

  await context.runCommand(`${binaryPath} -s stop`);

  const configContent = await context.readFile(configPath);
  const updatedConfig = enableVerboseLogging(configContent);
  await context.writeFile(configPath, updatedConfig);

  await context.runCommand(`${binaryPath} -s start`);

  const updatedLogsResult = await context.runCommand(`journalctl -u ${serviceName}`);

  return {
    initialLogs: initialLogsResult.stdout,
    updatedLogs: updatedLogsResult.stdout,
    updatedConfig,
  };
}
