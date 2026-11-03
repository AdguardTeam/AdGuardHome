import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@solidjs/testing-library';

import { CheckResult } from 'panel/components/UserRules/blocks/CheckResult/CheckResult';
import type { CheckResultData } from 'panel/components/UserRules/types';
import { FILTERED_STATUS, SPECIAL_FILTER_ID } from 'panel/helpers/constants';

vi.mock('panel/stores/filtering', () => ({
    filteringState: {
        filters: [],
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

    it('renders every rule returned by the backend without deduplication', () => {
        renderCheckResult({
            hostname: '10.10.10.10.in-addr.arpa',
            reason: FILTERED_STATUS.REWRITE_RULE,
            rules: Array.from({ length: 3 }, () => ({
                filter_list_id: SPECIAL_FILTER_ID.CUSTOM_FILTERING_RULES,
                text: ptrRule,
            })),
        });

        const ruleLines = screen.getAllByTestId('user-rules-result-rule');
        expect(ruleLines).toHaveLength(3);
        ruleLines.forEach((line) => expect(line).toHaveTextContent(ptrRule));
    });

    it('renders a single rule exactly once', () => {
        renderCheckResult({
            hostname: 'blocked.example',
            reason: FILTERED_STATUS.FILTERED_BLACK_LIST,
            rules: [{ filter_list_id: 101, text: '||blocked.example^' }],
        });

        const ruleLines = screen.getAllByTestId('user-rules-result-rule');
        expect(ruleLines).toHaveLength(1);
        expect(ruleLines[0]).toHaveTextContent('||blocked.example^');
    });
});
