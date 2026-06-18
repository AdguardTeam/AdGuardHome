import { execFile } from 'node:child_process';
import { promisify } from 'node:util';

const execFileAsync = promisify(execFile);

export interface CommandResult {
  stdout: string;
  stderr: string;
}

export type CommandRunner = (command: string, args: string[]) => Promise<CommandResult>;

export interface UninstallServiceOptions {
  binaryPath?: string;
  runner?: CommandRunner;
  sudo?: boolean;
}

const DEFAULT_BINARY_PATH = '/opt/AdGuardHome/AdGuardHome';

async function defaultRunner(command: string, args: string[]): Promise<CommandResult> {
  const result = await execFileAsync(command, args, {
    maxBuffer: 10 * 1024 * 1024,
  });
  return { stdout: result.stdout, stderr: result.stderr };
}

/**
 * Uninstalls AdGuardHome service by running the binary with `-s uninstall` flag.
 * @param options Configuration options for uninstalling the service.
 * @returns CommandResult from the execution.
 */
export async function uninstallAdGuardHomeService(options: UninstallServiceOptions = {}): Promise<CommandResult> {
  const binaryPath = options.binaryPath ?? DEFAULT_BINARY_PATH;
  const runner = options.runner ?? defaultRunner;
  const sudo = options.sudo ?? false;

  const command = sudo ? 'sudo' : binaryPath;
  const args = sudo ? [binaryPath, '-s', 'uninstall'] : ['-s', 'uninstall'];

  try {
    return await runner(command, args);
  } catch (error) {
    const details = error instanceof Error ? error.message : String(error);
    throw new Error(`Failed to uninstall AdGuardHome service: ${details}`);
  }
}
