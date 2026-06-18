import dgram from 'node:dgram';
import { setTimeout as delay } from 'node:timers/promises';
import * as dnsPacket from 'dns-packet';

export interface QueryDnsOptions {
  server?: string;
  port: number;
  name: string;
  type: string;
  timeoutMs?: number;
}

export interface QueryDnsWithRetryOptions extends QueryDnsOptions {
  attempts?: number;
  retryDelayMs?: number;
}

function normalizeDomainName(value: string): string {
  return value.endsWith('.') ? value : `${value}.`;
}

export async function queryDns(options: QueryDnsOptions): Promise<any> {
  const server = options.server ?? '127.0.0.1';
  const timeoutMs = options.timeoutMs ?? 2_000;
  const id = Math.floor(Math.random() * 65_535);
  const socket = dgram.createSocket(server.includes(':') ? 'udp6' : 'udp4');

  return new Promise((resolve, reject) => {
    let settled = false;

    const finish = (callback: () => void) => {
      if (settled) {
        return;
      }

      settled = true;
      clearTimeout(timer);
      socket.removeAllListeners();
      socket.close();
      callback();
    };

    const timer = setTimeout(() => {
      finish(() => reject(new Error(`DNS query timed out for ${options.name} (${options.type}) on port ${options.port}`)));
    }, timeoutMs);

    socket.on('error', (error) => {
      finish(() => reject(error));
    });

    socket.on('message', (message) => {
      try {
        const packet = dnsPacket.decode(message);
        if (packet.id !== id || packet.type !== 'response') {
          return;
        }

        finish(() => resolve(packet));
      } catch (error) {
        finish(() => reject(error));
      }
    });

    const payload = dnsPacket.encode({
      type: 'query',
      id,
      flags: dnsPacket.RECURSION_DESIRED,
      questions: [
        {
          name: options.name,
          type: options.type,
        },
      ],
    });

    socket.send(payload, options.port, server, (error) => {
      if (error) {
        finish(() => reject(error));
      }
    });
  });
}

export async function queryDnsWithRetry(options: QueryDnsWithRetryOptions): Promise<any> {
  const attempts = options.attempts ?? 3;
  const retryDelayMs = options.retryDelayMs ?? 1_000;
  let lastError: Error | undefined;

  for (let attempt = 1; attempt <= attempts; attempt += 1) {
    try {
      return await queryDns(options);
    } catch (error) {
      lastError = error instanceof Error ? error : new Error(String(error));
      if (attempt < attempts) {
        await delay(retryDelayMs);
      }
    }
  }

  throw lastError ?? new Error(`DNS query failed for ${options.name} (${options.type})`);
}

export function formatDnsAnswer(answer: any): string {
  switch (answer.type) {
    case 'A':
    case 'AAAA':
      return String(answer.data);
    case 'NS':
    case 'CNAME':
    case 'PTR':
      return normalizeDomainName(String(answer.data));
    case 'TXT':
      if (Array.isArray(answer.data)) {
        return answer.data
          .map((chunk) => Buffer.isBuffer(chunk) ? chunk.toString('utf8') : String(chunk))
          .join('');
      }
      return String(answer.data);
    case 'SRV':
      return `${answer.data.priority} ${answer.data.weight} ${answer.data.port} ${normalizeDomainName(answer.data.target)}`;
    case 'SOA':
      return [
        normalizeDomainName(answer.data.mname),
        normalizeDomainName(answer.data.rname),
        answer.data.serial,
        answer.data.refresh,
        answer.data.retry,
        answer.data.expire,
        answer.data.minimum,
      ].join(' ');
    default:
      return String(answer.data);
  }
}

export function formatDnsAnswers(packet: any): string[] {
  return (packet.answers ?? []).map((answer: any) => formatDnsAnswer(answer));
}
