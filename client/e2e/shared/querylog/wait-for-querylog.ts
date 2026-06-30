import { waitFor } from '../polling/retry.ts';
import { jsonRequest, type JsonFetchLike } from '../api/json-client.ts';
import type { QueryLogAnswer, QueryLogRecord } from './types.ts';
export type { QueryLogAnswer, QueryLogRecord } from './types.ts';

interface QueryLogResponse {
  data?: QueryLogRecord[];
}

function normalizeHost(host?: string): string | undefined {
  return host?.replace(/\.$/, '').toLowerCase();
}

export interface QueryLogRecordMatch {
  domain?: string;
  client?: string;
  type?: string;
  status?: string;
  answerValue?: string;
  afterTimeMs?: number;
}

function matchesQueryLogRecord(record: QueryLogRecord, match: QueryLogRecordMatch): boolean {
  const hostFromQuestion = normalizeHost(record.question?.host || record.question?.name);
  const hostFromQhost = normalizeHost(record.qhost);
  const expectedDomain = match.domain === undefined ? undefined : normalizeHost(match.domain);
  const matchesDomain = expectedDomain === undefined
    || hostFromQuestion === expectedDomain
    || hostFromQhost === expectedDomain;
  const matchesClient = match.client === undefined || record.client === match.client;
  const matchesType = match.type === undefined || record.question?.type === match.type;
  const matchesStatus = match.status === undefined || record.status === match.status;
  const matchesAnswerValue = match.answerValue === undefined
    || (record.answer ?? []).some((answer) => answer.value === match.answerValue)
    || (record.original_answer ?? []).some((answer) => answer.value === match.answerValue);
  const matchesTime = match.afterTimeMs === undefined
    || (record.time !== undefined && new Date(record.time).getTime() > match.afterTimeMs);

  return matchesDomain && matchesClient && matchesType && matchesStatus && matchesAnswerValue && matchesTime;
}

export async function waitForMatchingQueryLogRecord(
  baseUrl: string,
  match: QueryLogRecordMatch,
  options: {
    fetchImpl?: JsonFetchLike;
    timeoutMs?: number;
    intervalMs?: number;
    path?: string;
  } = {},
): Promise<QueryLogRecord> {
  return waitFor(async () => {
    const payload = await jsonRequest<QueryLogResponse>({
      baseUrl,
      path: options.path ?? '/control/querylog',
      method: 'GET',
      fetchImpl: options.fetchImpl,
      headers: {
        Accept: 'application/json',
      },
    });

    const records = payload.data ?? [];
    return records.find((record) => matchesQueryLogRecord(record, match));
  }, {
    timeoutMs: options.timeoutMs ?? 10_000,
    intervalMs: options.intervalMs ?? 500,
  });
}

export async function waitForQueryLogRecord(
  baseUrl: string,
  domain: string,
  options: {
    fetchImpl?: JsonFetchLike;
    timeoutMs?: number;
    intervalMs?: number;
    path?: string;
    client?: string;
    type?: string;
    status?: string;
    answerValue?: string;
    afterTimeMs?: number;
  } = {},
): Promise<QueryLogRecord> {
  return waitForMatchingQueryLogRecord(baseUrl, {
    domain,
    client: options.client,
    type: options.type,
    status: options.status,
    answerValue: options.answerValue,
    afterTimeMs: options.afterTimeMs,
  }, options);
}
