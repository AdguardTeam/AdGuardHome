import { type CommandResult, type CommandRunner, defaultRunner } from './checkRunning.ts';

/**
 * Installs AdGuardHome as a service.
 * @param runner The command runner to use.
 * @param path The path to the AdGuardHome binary (default: ./AdGuardHome).
 */
export async function installService(runner: CommandRunner = defaultRunner, path: string = './AdGuardHome'): Promise<CommandResult> {
  return runner(`sudo ${path} -s install`);
}
