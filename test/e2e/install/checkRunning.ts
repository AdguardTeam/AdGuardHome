import { exec } from 'node:child_process';
import { promisify } from 'node:util';

const execAsync = promisify(exec);

export interface CommandResult {
  stdout: string;
  stderr: string;
}

export type CommandRunner = (command: string) => Promise<CommandResult>;

export const defaultRunner: CommandRunner = async (command: string) => {
  return execAsync(command);
};

/**
 * Checks if AdGuardHome process is running by checking the process list.
 * @param runner The command runner to use (defaults to child_process.exec).
 * @returns true if AdGuardHome is running, false otherwise.
 */
export async function checkAdGuardHomeRunning(runner: CommandRunner = defaultRunner): Promise<boolean> {
  try {
    // Check for AdGuardHome process, excluding the grep process itself.
    // We use 'grep -F -e' as per the test case description, but also filter out grep to avoid false positives.
    // However, since we are running in a shell, the grep process might appear.
    // A common trick is `grep [A]dGuardHome` or checking the process name directly.
    // Here we will run the command and filter the output in JS.
    const { stdout } = await runner("ps aux | grep -F -e 'AdGuardHome'");

    const lines = stdout.split('\n');
    // Filter lines that contain 'AdGuardHome' but do not contain the grep command itself (which might appear if we grep for it).
    // The grep process usually contains 'grep' in its command line.
    const runningProcesses = lines.filter(line =>
      line.includes('AdGuardHome') && !line.includes('grep')
    );

    return runningProcesses.length > 0;
  } catch (error) {
    // If the command fails (e.g., ps not found, or grep returns exit code 1 if no match found), return false.
    return false;
  }
}

/**
 * Checks if a port is in use using lsof.
 * @param port The port number to check.
 * @param runner The command runner to use (defaults to child_process.exec).
 * @returns true if the port is in use, false otherwise.
 */
export async function checkPortInUse(port: number, runner: CommandRunner = defaultRunner): Promise<boolean> {
  try {
    // sudo is recommended in the test case, but for automated tests we might not have sudo or need it if checking user processes.
    // We will try running lsof without sudo first. If it fails or returns nothing, we assume not running or not detectable.
    // The test case says "Run sudo lsof -i :3000".
    // We'll run `lsof -i :<port>` and check output.
    const { stdout } = await runner(`lsof -i :${port}`);
    return stdout.trim().length > 0;
  } catch (error) {
    return false;
  }
}
