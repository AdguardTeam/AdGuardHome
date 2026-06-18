import { execFile } from 'node:child_process';
import { promisify } from 'node:util';

const execFileAsync = promisify(execFile);

export interface CommandResult {
  stdout: string;
  stderr: string;
}

export type CommandRunner = (command: string, args: string[]) => Promise<CommandResult>;

export interface ReinstallViaCurlOptions {
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
    throw new Error(`AdGuard reinstall script URL must use HTTPS: ${resolved}`);
  }

  return resolved;
}

export async function reinstallAdGuardHomeSameVersionViaCurl(
  options: ReinstallViaCurlOptions = {},
): Promise<CommandResult[]> {
  const scriptUrl = normalizeScriptUrl(options.scriptUrl);
  const runner = options.runner ?? defaultRunner;
  const command = `curl -s -S -L ${scriptUrl} | sh -s -- -r -v`;
  const results: CommandResult[] = [];

  for (let attempt = 1; attempt <= 2; attempt += 1) {
    try {
      const result = await runner('bash', ['-lc', command]);
      results.push(result);
    } catch (error) {
      const details = error instanceof Error ? error.message : String(error);
      throw new Error(`Failed to reinstall AdGuardHome via curl on attempt ${attempt}: ${details}`);
    }
  }

  return results;
}
