import assert from 'node:assert/strict';
import type { QueryLogAnswer, QueryLogRecord } from '../shared/querylog/types.ts';

export type FetchLike = typeof fetch;
export type { QueryLogAnswer, QueryLogRecord } from '../shared/querylog/types.ts';

export interface CustomRuleTestCase {
  name: string;
  customRule: string | string[];
  query: {
    domain: string;
    type: string;
  };
  expected: {
    originalAnswerContains?: string;
    originalAnswerValues?: string[];
    answerContains?: string;
    answerValues?: string[];
    status?: string;
    emptyAnswer?: boolean;
    notEmptyAnswer?: boolean;
    reason?: string;
    rule?: string;
  };
}

interface QueryLogResponse {
  data?: QueryLogRecord[];
}

export interface CustomRuleCheckContext {
  adGuardBaseUrl: string;
  runDnsLookup: (query: CustomRuleTestCase['query']) => Promise<unknown>;
  fetchImpl?: FetchLike;
  now?: () => number;
}

function formatUnknownError(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

function normalizeDnsName(value?: string): string | undefined {
  return value?.replace(/\.$/, '').toLowerCase();
}

function normalizeDnsType(value?: string): string | undefined {
  return value?.toUpperCase();
}

function getQueryLogRecordTimeMs(record: QueryLogRecord): number | undefined {
  if (!record.time) {
    return undefined;
  }

  const timeMs = Date.parse(record.time);
  return Number.isNaN(timeMs) ? undefined : timeMs;
}

function matchesQueryLogRecord(record: QueryLogRecord, query: CustomRuleTestCase['query']): boolean {
  const recordHost = normalizeDnsName(record.question?.host || record.question?.name || record.qhost);
  const queryHost = normalizeDnsName(query.domain);

  if (recordHost !== queryHost) {
    return false;
  }

  const recordType = normalizeDnsType(record.question?.type);
  const queryType = normalizeDnsType(query.type);

  return recordType === undefined || recordType === queryType;
}

function findLatestMatchingQueryLogRecord(
  records: QueryLogRecord[],
  query: CustomRuleTestCase['query'],
  minRecordTimeMs: number,
): QueryLogRecord | undefined {
  const matchingRecords = records.filter((record) => matchesQueryLogRecord(record, query));
  const timedMatchingRecords = matchingRecords
    .map((record) => ({
      record,
      timeMs: getQueryLogRecordTimeMs(record),
    }))
    .filter((entry): entry is { record: QueryLogRecord; timeMs: number } => entry.timeMs !== undefined);

  const recentMatchingRecords = timedMatchingRecords
    .filter((entry) => entry.timeMs >= minRecordTimeMs)
    .sort((left, right) => right.timeMs - left.timeMs);

  if (recentMatchingRecords.length > 0) {
    return recentMatchingRecords[0].record;
  }

  if (timedMatchingRecords.length > 0) {
    return undefined;
  }

  return matchingRecords[0];
}

function queryLogRecordMatchesExpected(record: QueryLogRecord, expected: CustomRuleTestCase['expected']): boolean {
  if (expected.status && record.status !== expected.status) {
    return false;
  }

  if (expected.reason && record.reason !== expected.reason) {
    return false;
  }

  if (expected.rule && record.rule !== expected.rule) {
    return false;
  }

  const answerValues = normalizeAnswerValues(getAnswerValues(record.answer));
  const originalAnswerValues = normalizeAnswerValues(getAnswerValues(record.original_answer));

  if (expected.answerContains && !answerValues.includes(expected.answerContains)) {
    return false;
  }

  if (expected.answerValues) {
    const expectedAnswers = normalizeAnswerValues(expected.answerValues);
    if (answerValues.join('\n') !== expectedAnswers.join('\n')) {
      return false;
    }
  }

  if (expected.originalAnswerContains && !originalAnswerValues.includes(expected.originalAnswerContains)) {
    return false;
  }

  if (expected.originalAnswerValues) {
    const expectedOriginalAnswers = normalizeAnswerValues(expected.originalAnswerValues);
    if (originalAnswerValues.join('\n') !== expectedOriginalAnswers.join('\n')) {
      return false;
    }
  }

  if (expected.emptyAnswer && answerValues.length !== 0) {
    return false;
  }

  if (expected.notEmptyAnswer && answerValues.length === 0) {
    return false;
  }

  return true;
}

function findBestMatchingQueryLogRecord(
  records: QueryLogRecord[],
  query: CustomRuleTestCase['query'],
  minRecordTimeMs: number,
  expected: CustomRuleTestCase['expected'],
): QueryLogRecord | undefined {
  const latestRecord = findLatestMatchingQueryLogRecord(records, query, minRecordTimeMs);
  if (!latestRecord) {
    return undefined;
  }

  const recentRecords = records
    .filter((record) => matchesQueryLogRecord(record, query))
    .filter((record) => {
      const timeMs = getQueryLogRecordTimeMs(record);
      return timeMs !== undefined && timeMs >= minRecordTimeMs;
    })
    .sort((left, right) => (getQueryLogRecordTimeMs(right) ?? 0) - (getQueryLogRecordTimeMs(left) ?? 0));

  // Return only the record that matches the expectation, so the caller keeps
  // polling (the query log is eventually-consistent) instead of settling on a
  // query-matching-but-not-yet-expected record.
  const exactRecentRecord = recentRecords.find((record) => queryLogRecordMatchesExpected(record, expected));
  return exactRecentRecord;
}

function getAnswerValues(answers: QueryLogAnswer[] = []): string[] {
  return answers.flatMap((answer) => typeof answer.value === 'string' ? [answer.value] : []);
}

function normalizeAnswerValues(values: string[]): string[] {
  return [...new Set(values)].sort((left, right) => left.localeCompare(right));
}

/**
 * Universal scenario runner for custom DNS rule validation.
 *
 * Flow:
 * 1) Add custom rule.
 * 2) Execute dnslookup request from test case.
 * 3) Assert query log includes original DNS answer with expected value.
 */
export async function runCustomRuleTestCase(
  testCase: CustomRuleTestCase,
  context: CustomRuleCheckContext,
): Promise<void> {
  const fetchImpl = context.fetchImpl ?? fetch;
  const now = context.now ?? Date.now;
  const queryStartedAtMs = now();

  const rules = Array.isArray(testCase.customRule) ? testCase.customRule : [testCase.customRule];

  const setRulesResponse = await fetchImpl(`${context.adGuardBaseUrl}/control/filtering/set_rules`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      rules,
    }),
  });

  assert.equal(
    setRulesResponse.ok,
    true,
    `Failed to set custom rule for case "${testCase.name}". Status: ${setRulesResponse.status}`,
  );

  let dnsLookupResult: unknown;
  let dnsLookupError: unknown;
  try {
    dnsLookupResult = await context.runDnsLookup(testCase.query);
  } catch (error) {
    dnsLookupError = error;
  }

  let matchedRecord: QueryLogRecord | undefined;
  const startTime = now();
  const timeoutMs = 10000;

  while (now() - startTime < timeoutMs) {
    const queryLogResponse = await fetchImpl(`${context.adGuardBaseUrl}/control/querylog`, {
      method: 'GET',
      headers: {
        Accept: 'application/json',
      },
    });

    if (queryLogResponse.ok) {
        const payload = (await queryLogResponse.json()) as QueryLogResponse;
        const records = payload.data ?? [];

        matchedRecord = findBestMatchingQueryLogRecord(records, testCase.query, queryStartedAtMs, testCase.expected);

        if (matchedRecord) {
            break;
        }
    }

    await new Promise(resolve => setTimeout(resolve, 500));
  }

  assert.ok(
    matchedRecord,
    [
      `Query log does not contain request for ${testCase.query.domain} in case "${testCase.name}" after ${timeoutMs}ms`,
      dnsLookupResult === undefined ? undefined : `DNS lookup result: ${JSON.stringify(dnsLookupResult)}`,
      dnsLookupError === undefined ? undefined : `DNS lookup error: ${formatUnknownError(dnsLookupError)}`,
    ].filter(Boolean).join('\n'),
  );

  if (testCase.expected.status) {
    assert.equal(
        matchedRecord?.status,
        testCase.expected.status,
        `Query log status mismatch for case "${testCase.name}". Expected: ${testCase.expected.status}, Actual: ${matchedRecord?.status}`
    );
  }

  if (testCase.expected.originalAnswerContains) {
    const originalAnswers = matchedRecord?.original_answer ?? [];
    const containsExpectedAnswer = originalAnswers.some(
      (answer) => answer.value === testCase.expected.originalAnswerContains,
    );

    assert.equal(
      containsExpectedAnswer,
      true,
      [
        `Query log does not contain expected original answer for case "${testCase.name}"`,
        `Expected value: ${testCase.expected.originalAnswerContains}`,
        `Actual original_answer: ${JSON.stringify(originalAnswers)}`,
      ].join('\n'),
    );
  }

  if (testCase.expected.originalAnswerValues) {
    const originalAnswers = normalizeAnswerValues(getAnswerValues(matchedRecord?.original_answer));
    const expectedOriginalAnswers = normalizeAnswerValues(testCase.expected.originalAnswerValues);

    assert.deepEqual(
      originalAnswers,
      expectedOriginalAnswers,
      [
        `Query log original answers mismatch for case "${testCase.name}"`,
        `Expected values: ${JSON.stringify(expectedOriginalAnswers)}`,
        `Actual original_answer: ${JSON.stringify(originalAnswers)}`,
      ].join('\n'),
    );
  }

  if (testCase.expected.answerContains) {
    const answers = matchedRecord?.answer ?? [];
    const containsExpectedAnswer = answers.some(
      (answer) => answer.value === testCase.expected.answerContains,
    );

    assert.equal(
      containsExpectedAnswer,
      true,
      [
        `Query log does not contain expected answer for case "${testCase.name}"`,
        `Expected value: ${testCase.expected.answerContains}`,
        `Actual answer: ${JSON.stringify(answers)}`,
      ].join('\n'),
    );
  }

  if (testCase.expected.answerValues) {
    const answers = normalizeAnswerValues(getAnswerValues(matchedRecord?.answer));
    const expectedAnswers = normalizeAnswerValues(testCase.expected.answerValues);

    assert.deepEqual(
      answers,
      expectedAnswers,
      [
        `Query log answer values mismatch for case "${testCase.name}"`,
        `Expected values: ${JSON.stringify(expectedAnswers)}`,
        `Actual answer: ${JSON.stringify(answers)}`,
      ].join('\n'),
    );
  }

  if (testCase.expected.emptyAnswer) {
      const answers = matchedRecord?.answer ?? [];
      assert.equal(
          answers.length,
          0,
          `Query log should have empty answer for case "${testCase.name}", but got ${answers.length} answers: ${JSON.stringify(answers)}`
      );
  }

  if (testCase.expected.notEmptyAnswer) {
      const answers = matchedRecord?.answer ?? [];
      assert.notEqual(
          answers.length,
          0,
          `Query log should NOT have empty answer for case "${testCase.name}"`
      );
  }

  if (testCase.expected.reason) {
    assert.equal(
      matchedRecord?.reason,
      testCase.expected.reason,
      `Query log reason mismatch for case "${testCase.name}"`,
    );
  }

  if (testCase.expected.rule) {
    assert.equal(
      matchedRecord?.rule,
      testCase.expected.rule,
      `Query log rule mismatch for case "${testCase.name}"`,
    );
  }
}
