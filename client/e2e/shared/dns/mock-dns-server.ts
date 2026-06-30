import dgram from 'node:dgram';
import * as dnsPacket from 'dns-packet';
import { Buffer } from 'node:buffer';

export interface DnsQuery {
  timestamp: number;
  id: number;
  question: string;
  type: string;
  remoteAddress: string;
  remotePort: number;
}

export interface MockDnsAnswer {
  name?: string;
  type: string;
  ttl?: number;
  data: unknown;
}

export interface ReservedUdpPort {
  port: number;
  socket: dgram.Socket;
}

function normalizeDnsName(value: string): string {
  return value.replace(/\.$/, '').toLowerCase();
}

export async function allocateUdpPort(host: string = '127.0.0.1'): Promise<ReservedUdpPort> {
  return new Promise((resolve, reject) => {
    const socket = dgram.createSocket(host.includes(':') ? 'udp6' : 'udp4');
    const onError = (error: Error) => {
      socket.close();
      reject(error);
    };
    socket.once('error', onError);
    socket.bind(0, host, () => {
      const address = socket.address();
      if (!address || typeof address === 'string') {
        socket.close(() => reject(new Error('Failed to allocate UDP port for mock DNS server')));
        return;
      }

      socket.removeListener('error', onError);
      resolve({
        port: address.port,
        socket,
      });
    });
  });
}

export class MockDnsServer {
  private server: dgram.Socket;
  private port: number;
  private reservedSocket: boolean;
  private queries: DnsQuery[] = [];
  private delayMs: number = 0;
  private pendingTimers = new Set<ReturnType<typeof setTimeout>>();
  private listening: boolean = false;
  private responses = new Map<string, MockDnsAnswer[]>();
  private nxDomains = new Set<string>();

  constructor(port: number | ReservedUdpPort) {
    if (typeof port === 'number') {
      this.port = port;
      this.server = dgram.createSocket('udp4');
      this.reservedSocket = false;
    } else {
      this.port = port.port;
      this.server = port.socket;
      this.reservedSocket = true;
      this.listening = true;
    }

    this.server.on('error', (err) => {
      console.error(`[MockDNS:${this.port}] Server error:\n${err.stack}`);
      this.server.close();
      this.listening = false;
    });

    this.server.on('message', (msg, rinfo) => {
      this.handleMessage(msg, rinfo);
    });

    this.server.on('listening', () => {
      this.listening = true;
    });
  }

  private handleMessage(msg: Buffer, rinfo: dgram.RemoteInfo) {
    console.log(`[MockDNS:${this.port}] Received message from ${rinfo.address}:${rinfo.port} size=${msg.length}`);
    let packet;
    try {
      packet = dnsPacket.decode(msg);
    } catch (e) {
      console.error(`[MockDNS:${this.port}] Failed to decode packet from ${rinfo.address}:${rinfo.port}`);
      return;
    }

    const question = packet.questions?.[0];
    console.log(`[MockDNS:${this.port}] Question: ${question?.name} ${question?.type}`);
    if (!question) return;

    const query: DnsQuery = {
      timestamp: Date.now(),
      id: packet.id!, // id is usually present
      question: question.name,
      type: question.type,
      remoteAddress: rinfo.address,
      remotePort: rinfo.port,
    };

    this.queries.push(query);

    if (this.delayMs > 0) {
      const timer = setTimeout(() => {
        this.pendingTimers.delete(timer);
        if (this.listening) this.sendResponse(packet, rinfo);
      }, this.delayMs);
      this.pendingTimers.add(timer);
    } else {
      this.sendResponse(packet, rinfo);
    }
  }

  private responseKey(name: string, type: string): string {
    return `${normalizeDnsName(name)}|${type.toUpperCase()}`;
  }

  private getAnswers(question: { name: string; type: string }): MockDnsAnswer[] {
    const configuredAnswers = this.responses.get(this.responseKey(question.name, question.type));
    if (configuredAnswers) {
      return configuredAnswers.map((answer) => ({
        ...answer,
        name: answer.name ?? question.name,
        ttl: answer.ttl ?? 600,
      }));
    }

    switch (question.type) {
      case 'A':
        return [{
          name: question.name,
          type: 'A',
          ttl: 600,
          data: '1.2.3.4',
        }];
      case 'AAAA':
        return [{
          name: question.name,
          type: 'AAAA',
          ttl: 600,
          data: '2001:db8::1',
        }];
      case 'SOA':
        return [{
          name: question.name,
          type: 'SOA',
          ttl: 600,
          data: {
            mname: 'ns1.example.com',
            rname: 'admin.example.com',
            serial: 2023010101,
            refresh: 3600,
            retry: 600,
            expire: 86400,
            minimum: 3600,
          },
        }];
      case 'NS':
        return [{
          name: question.name,
          type: 'NS',
          ttl: 600,
          data: 'ns1.example.com',
        }];
      default:
        return [];
    }
  }

  private sendResponse(packet: any, rinfo: dgram.RemoteInfo) {
    const question = packet.questions?.[0];
    const isNxDomain = question && this.nxDomains.has(this.responseKey(question.name, question.type));
    const answers = (!isNxDomain && question) ? this.getAnswers(question) : [];

    // dns-packet encodes rcode in the lower 4 bits of the flags field.
    // NXDOMAIN = rcode 3. `rcode` property in encode() is silently ignored.
    const rcodeFlags = isNxDomain ? 3 : 0;
    const response = dnsPacket.encode({
      type: 'response',
      id: packet.id,
      flags: dnsPacket.AUTHORITATIVE_ANSWER | rcodeFlags,
      questions: packet.questions,
      answers: answers,
    });

    this.server.send(response, rinfo.port, rinfo.address, (err) => {
      if (err) {
        console.error(`[MockDNS:${this.port}] Failed to send response: ${err}`);
      }
    });
  }

  public async start(): Promise<void> {
    if (this.reservedSocket) {
      console.log(`[MockDNS:${this.port}] Server already bound and listening`);
      return;
    }

    return new Promise((resolve, reject) => {
      this.server.bind(this.port, () => {
        console.log(`[MockDNS:${this.port}] Server bound and listening`);
        resolve();
      });
      // Handle bind errors if any
      this.server.once('error', (err) => {
          if (!this.listening) reject(err);
      });
    });
  }

  public async stop(): Promise<void> {
    for (const timer of this.pendingTimers) clearTimeout(timer);
    this.pendingTimers.clear();
    return new Promise((resolve) => {
      if (!this.listening) {
          try {
              this.server.close();
          } catch(e) {}
          return resolve();
      }
      this.server.close(() => {
        this.listening = false;
        resolve();
      });
    });
  }

  public getQueries(): DnsQuery[] {
    return this.queries;
  }

  public clearQueries() {
    this.queries = [];
  }

  public setAnswers(name: string, type: string, answers: MockDnsAnswer[]) {
    this.responses.set(this.responseKey(name, type), answers);
  }

  /** Make the mock respond with NXDOMAIN rcode for the given name+type. */
  public setNxDomain(name: string, type: string) {
    this.nxDomains.add(this.responseKey(name, type));
  }

  public getPort(): number {
    return this.port;
  }

  public setDelay(ms: number) {
    this.delayMs = ms;
  }
}
