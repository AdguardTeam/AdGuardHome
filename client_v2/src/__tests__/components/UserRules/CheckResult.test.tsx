import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@solidjs/testing-library';

import { CheckResult } from 'panel/components/UserRules/blocks/CheckResult/CheckResult';
import type { CheckResultData } from 'panel/components/UserRules/types';
import { FILTERED_STATUS, SPECIAL_FILTER_ID } from 'panel/helpers/constants';
import { copy, copyInDom } from 'panel/__tests__/helpers/copy';

vi.mock('panel/stores/filtering', () => ({
    filteringState: {
        filters: [
            {
                id: 1793714189,
                name: 'Multi list',
                url: 'https://filters.example/multi.txt',
                enabled: true,
                lastUpdated: '',
                rulesCount: 2,
            },
        ],
        whitelistFilters: [],
    },
}));

vi.mock('panel/stores/services', () => ({
    servicesState: {
        allServices: [],
    },
}));

vi.mock('panel/common/intl', async () =>
    (await import('panel/__tests__/helpers/copy')).createIntlMock(),
);

const renderCheckResult = (checkResult: CheckResultData) =>
    render(() => (
        <CheckResult
            checkResult={checkResult}
            processingRules={false}
            onAction={vi.fn()}
            onEditRewrite={vi.fn()}
            onDeleteRewrite={vi.fn()}
        />
    ));

describe('CheckResult rules', () => {
    const ptrRule = '||10.10.10.10.in-addr.arpa^$dnsrewrite=NOERROR;PTR;router.home.arpa.';

    it('renders every rule returned by the backend under one source label', () => {
        renderCheckResult({
            hostname: '10.10.10.10.in-addr.arpa',
            reason: FILTERED_STATUS.REWRITE_RULE,
            rules: Array.from({ length: 3 }, () => ({
                filter_list_id: SPECIAL_FILTER_ID.CUSTOM_FILTERING_RULES,
                text: ptrRule,
            })),
        });

        const sourceLabels = screen.getAllByTestId('user-rules-result-source');
        expect(sourceLabels).toHaveLength(1);
        expect(sourceLabels[0]).toHaveTextContent(
            copyInDom('user_rules_source', { value: copy('custom_rules') }),
        );

        const ruleLines = screen.getAllByTestId('user-rules-result-rule');
        expect(ruleLines).toHaveLength(3);
        ruleLines.forEach((line) => expect(line).toHaveTextContent(ptrRule));
    });

    it('renders each rule under its own filter list source', () => {
        renderCheckResult({
            hostname: 'hosts.example',
            reason: FILTERED_STATUS.FILTERED_BLACK_LIST,
            rules: [
                {
                    filter_list_id: SPECIAL_FILTER_ID.CUSTOM_FILTERING_RULES,
                    text: '127.0.0.1 hosts.example',
                },
                {
                    filter_list_id: 1793714189,
                    text: '0.0.0.0 hosts.example',
                },
            ],
        });

        const sourceLabels = screen.getAllByTestId('user-rules-result-source');
        expect(sourceLabels).toHaveLength(2);
        expect(sourceLabels[0]).toHaveTextContent(
            copyInDom('user_rules_source', { value: copy('custom_rules') }),
        );
        expect(sourceLabels[1]).toHaveTextContent(
            copyInDom('user_rules_source_blocklist', { value: 'Multi list' }),
        );

        const ruleLines = screen.getAllByTestId('user-rules-result-rule');
        expect(ruleLines).toHaveLength(2);
        expect(sourceLabels[0].nextElementSibling).toHaveTextContent('127.0.0.1 hosts.example');
        expect(sourceLabels[1].nextElementSibling).toHaveTextContent('0.0.0.0 hosts.example');
    });

    it('renders a rule without a filter list id without a source label', () => {
        renderCheckResult({
            hostname: 'custom.example',
            reason: FILTERED_STATUS.FILTERED_BLACK_LIST,
            rules: [{ text: '||custom.example^' }],
        });

        expect(screen.queryAllByTestId('user-rules-result-source')).toHaveLength(0);
        const ruleLines = screen.getAllByTestId('user-rules-result-rule');
        expect(ruleLines).toHaveLength(1);
        expect(ruleLines[0]).toHaveTextContent('||custom.example^');
    });

    it('skips an empty-text rule in a Safe Search result', () => {
        renderCheckResult({
            hostname: 'www.google.com',
            reason: FILTERED_STATUS.FILTERED_SAFE_SEARCH,
            rules: [{ filter_list_id: SPECIAL_FILTER_ID.SAFE_SEARCH, text: '' }],
            cname: 'forcesafesearch.google.com',
        });

        expect(screen.queryAllByTestId('user-rules-result-rule')).toHaveLength(0);
        expect(screen.queryAllByTestId('user-rules-result-source')).toHaveLength(0);
    });

    it('hides source and rule lines for a parental control result', () => {
        renderCheckResult({
            hostname: '4chan.org',
            reason: FILTERED_STATUS.FILTERED_PARENTAL,
            rules: [
                {
                    filter_list_id: SPECIAL_FILTER_ID.PARENTAL,
                    text: 'parental CATEGORY_BLACKLISTED',
                },
            ],
        });

        expect(screen.queryAllByTestId('user-rules-result-source')).toHaveLength(0);
        expect(screen.queryAllByTestId('user-rules-result-rule')).toHaveLength(0);
        expect(
            screen.getByText(
                copyInDom('user_rules_reason', {
                    reason: copy('user_rules_reason_blocked_by', {
                        source: copy('parental_control'),
                    }),
                }),
            ),
        ).toBeInTheDocument();
    });

    it('keeps source and rule lines for a safe browsing result', () => {
        renderCheckResult({
            hostname: 'malware.example',
            reason: FILTERED_STATUS.FILTERED_SAFE_BROWSING,
            rules: [
                { filter_list_id: SPECIAL_FILTER_ID.SAFE_BROWSING, text: 'adguard-malware-shavar' },
            ],
        });

        const sourceLabels = screen.getAllByTestId('user-rules-result-source');
        expect(sourceLabels).toHaveLength(1);
        expect(sourceLabels[0]).toHaveTextContent(
            copyInDom('user_rules_source', { value: copy('safe_browsing') }),
        );

        const ruleLines = screen.getAllByTestId('user-rules-result-rule');
        expect(ruleLines).toHaveLength(1);
        expect(ruleLines[0]).toHaveTextContent('adguard-malware-shavar');
    });

    it('renders a single rule exactly once', () => {
        renderCheckResult({
            hostname: 'blocked.example',
            reason: FILTERED_STATUS.FILTERED_BLACK_LIST,
            rules: [{ filter_list_id: 1793714189, text: '||blocked.example^' }],
        });

        const ruleLines = screen.getAllByTestId('user-rules-result-rule');
        expect(ruleLines).toHaveLength(1);
        expect(ruleLines[0]).toHaveTextContent('||blocked.example^');
    });
});
