import { execFile } from 'node:child_process';
import { promisify } from 'node:util';

const execFileAsync = promisify(execFile);

export interface CommandResult {
  stdout: string;
  stderr: string;
}

export type CommandRunner = (command: string, args: string[]) => Promise<CommandResult>;

export type ServiceAction = 'start' | 'stop' | 'restart' | 'status' | 'reload';

export interface RunServiceActionOptions {
  binaryPath?: string;
  runner?: CommandRunner;
  sudo?: boolean;
}

const DEFAULT_BINARY_PATH = './AdGuardHome';

async function defaultRunner(command: string, args: string[]): Promise<CommandResult> {
  const result = await execFileAsync(command, args, {
    maxBuffer: 10 * 1024 * 1024,
  });

  return {
    stdout: result.stdout,
    stderr: result.stderr,
  };
}

/**
 * Executes an AdGuard Home service action with strict action validation.
 */
export async function runServiceAction(
  action: ServiceAction,
  options: RunServiceActionOptions = {},
): Promise<CommandResult> {
  const binaryPath = options.binaryPath ?? DEFAULT_BINARY_PATH;
  const runner = options.runner ?? defaultRunner;
  const sudo = options.sudo ?? false;

  const command = sudo ? 'sudo' : binaryPath;
  const args = sudo ? [binaryPath, '-s', action] : ['-s', action];

  try {
    return await runner(command, args);
  } catch (error) {
    const details = error instanceof Error ? error.message : String(error);
    throw new Error(`Failed to execute service action "${action}": ${details}`);
  }
}

/**
 * Parses AdGuard Home service status from command output.
 */
export function parseServiceStatus(output: string): 'running' | 'stopped' | 'unknown' {
  const normalized = output.toLowerCase();

  if (normalized.includes('service is running')) {
    return 'running';
  }

  if (normalized.includes('service is stopped')) {
    return 'stopped';
  }

  return 'unknown';
}

/**
 * Convenience helper for testcase 4040 to keep repeated flow modular.
 */
export async function runStopStartFlow(
  options: RunServiceActionOptions = {},
): Promise<{ stop: CommandResult; start: CommandResult }> {
  const stop = await runServiceAction('stop', options);
  const start = await runServiceAction('start', options);

  return { stop, start };
}
