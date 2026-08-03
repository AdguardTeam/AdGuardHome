import { render } from '@solidjs/testing-library';
import { describe, expect, test } from 'vitest';

import intl from 'panel/common/intl';
import { StatusCell } from 'panel/components/QueryLog/blocks/LogTable/blocks/StatusCell';
import type { NormalizedQueryLogItem } from 'panel/helpers/helpers';
import theme from 'panel/lib/theme';

describe('StatusCell', () => {
    test('presents a failed unfiltered response as an error', () => {
        const row: NormalizedQueryLogItem = {
            time: '2026-08-03T00:00:00Z',
            domain: 'example.org',
            unicodeName: 'example.org',
            type: 'A',
            response: [],
            reason: 'NotFilteredNotFound',
            client: '192.0.2.1',
            client_info: null,
            rules: [],
            status: 'SERVFAIL',
            originalResponse: [],
            tracker: null,
            elapsedMs: '1',
        };

        const { getByText, queryByText } = render(() => <StatusCell row={row} />);
        const errorStatus = getByText(intl.getMessage('error'));

        expect(errorStatus).toHaveClass(theme.status.statusRed);
        expect(queryByText(intl.getMessage('query_log_processed'))).not.toBeInTheDocument();
    });
});
