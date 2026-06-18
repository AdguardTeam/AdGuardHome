import { execFile } from 'node:child_process';
import { promisify } from 'node:util';

const execFileAsync = promisify(execFile);

export interface CommandResult {
  stdout: string;
  stderr: string;
}

export type CommandRunner = (command: string, args: string[]) => Promise<CommandResult>;

export interface UninstallViaCurlOptions {
  scriptUrl?: string;
  runner?: CommandRunner;
}

const INSTALL_SCRIPT_URL =
  process.env.ADGUARD_INSTALL_SCRIPT_URL || 'https://raw.githubusercontent.com/AdguardTeam/AdGuardHome/master/scripts/install.sh';

async function defaultRunner(command: string, args: string[]): Promise<CommandResult> {
  const result = await execFileAsync(command, args, {
    maxBuffer: 10 * 1024 * 1024,
  });

  return {
    stdout: result.stdout,
    stderr: result.stderr,
  };
}

function normalizeScriptUrl(scriptUrl?: string): string {
  const resolved = scriptUrl?.trim() || INSTALL_SCRIPT_URL;
  if (!resolved.startsWith('https://')) {
    throw new Error(`AdGuard uninstall script URL must use HTTPS: ${resolved}`);
  }

  return resolved;
}

export async function uninstallAdGuardHomeViaCurl(options: UninstallViaCurlOptions = {}): Promise<CommandResult> {
  const scriptUrl = normalizeScriptUrl(options.scriptUrl);
  const runner = options.runner ?? defaultRunner;
  const command = `curl -s -S -L ${scriptUrl} | sh -s -- -u -v`;

  try {
    return await runner('bash', ['-lc', command]);
  } catch (error) {
    const details = error instanceof Error ? error.message : String(error);
    throw new Error(`Failed to uninstall AdGuardHome via curl: ${details}`);
  }
}
