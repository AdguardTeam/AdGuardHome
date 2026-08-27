import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { ComponentProps } from 'solid-js';
import { render, screen, fireEvent } from '@solidjs/testing-library';

import intl from 'panel/common/intl';
import { ActionFooter } from 'panel/components/QueryLog/blocks/DetailModal/blocks/ActionFooter';
import { findRewriteRuleByDomain } from 'panel/stores/rewrites';
import type { Filter, NormalizedQueryLogItem } from 'panel/helpers/helpers';
import type { RewriteEntry } from 'panel/api/model/rewriteEntry';

// The rewrites store is module-scoped; mock the lookup so the footer's
// rewrite-rule detection is deterministic in tests.
vi.mock('panel/stores/rewrites', () => ({
    findRewriteRuleByDomain: vi.fn(),
}));

const makeEntry = (overrides: Partial<NormalizedQueryLogItem> = {}): NormalizedQueryLogItem => ({
    time: '2026-08-25T10:00:00Z',
    domain: 'example.org',
    unicodeName: 'example.org',
    type: 'A',
    response: [],
    client: '192.168.0.40',
    client_info: null,
    rules: [],
    originalResponse: [],
    tracker: null,
    ...overrides,
});

const testFilter: Filter = {
    id: 5,
    name: 'AdGuard DNS Filter',
    url: 'https://example.org/filter.txt',
    enabled: true,
    lastUpdated: '',
    rulesCount: 100,
};

const defaultProps: ComponentProps<typeof ActionFooter> = {
    entry: makeEntry(),
    filters: [],
    onClose: vi.fn(),
    onBlock: vi.fn(),
    onAddToAllowlist: vi.fn(),
    onAllowService: vi.fn(),
    onDisableFilter: vi.fn(),
    onDisableSafeBrowsing: vi.fn(),
    onDisableParental: vi.fn(),
    onDisableSafeSearch: vi.fn(),
    onRemoveRewrite: vi.fn(),
    onEditRewrite: vi.fn(),
};

describe('ActionFooter', () => {
    beforeEach(() => {
        vi.spyOn(intl, 'getMessage').mockImplementation((key) => key);
        vi.mocked(findRewriteRuleByDomain).mockReset();
        vi.mocked(findRewriteRuleByDomain).mockReturnValue(undefined);
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    it('processed entry → Add to allowlist (primary) + Block (danger), in order', () => {
        render(() => (
            <ActionFooter {...defaultProps} entry={makeEntry({ reason: 'NotFilteredNotFound' })} />
        ));
        const footer = screen.getByTestId('query-log-detail-action-footer');
        const buttons = footer.querySelectorAll('button');
        expect(buttons).toHaveLength(2);
        expect(buttons[0]).toHaveAttribute('data-action', 'allowlist');
        expect(buttons[1]).toHaveAttribute('data-action', 'block');
    });

    it('clicking Block calls onBlock with the domain and onClose', () => {
        const props = {
            ...defaultProps,
            entry: makeEntry({ reason: 'NotFilteredNotFound' }),
        };
        render(() => <ActionFooter {...props} />);
        fireEvent.click(screen.getByTestId('query-log-detail-action-block'));
        expect(props.onBlock).toHaveBeenCalledWith('example.org');
        expect(props.onClose).toHaveBeenCalled();
    });

    it('clicking Add to allowlist calls onAddToAllowlist with the domain and onClose', () => {
        const props = {
            ...defaultProps,
            entry: makeEntry({ reason: 'NotFilteredNotFound' }),
        };
        render(() => <ActionFooter {...props} />);
        fireEvent.click(screen.getByTestId('query-log-detail-action-allowlist'));
        expect(props.onAddToAllowlist).toHaveBeenCalledWith('example.org');
        expect(props.onClose).toHaveBeenCalled();
    });

    it('blocked by filter → Disable filter resolves the filter from props', () => {
        const props = {
            ...defaultProps,
            entry: makeEntry({
                reason: 'FilteredBlackList',
                rules: [{ filter_list_id: 5 }],
            }),
            filters: [testFilter],
        };
        render(() => <ActionFooter {...props} />);
        const btn = screen.getByTestId('query-log-detail-action-disable-filter');
        expect(btn).toHaveAttribute('data-action', 'disable-filter');
        fireEvent.click(btn);
        expect(props.onDisableFilter).toHaveBeenCalledWith(testFilter);
        expect(props.onClose).toHaveBeenCalled();
    });

    it('blocked service → Allow service calls onAllowService with the service id and onClose', () => {
        const props = {
            ...defaultProps,
            entry: makeEntry({ reason: 'FilteredBlockedService', serviceName: 'amazon' }),
        };
        render(() => <ActionFooter {...props} />);
        fireEvent.click(screen.getByTestId('query-log-detail-action-allow-service'));
        expect(props.onAllowService).toHaveBeenCalledWith('amazon');
        expect(props.onClose).toHaveBeenCalled();
    });

    it('blocked threat → Disable browsing security calls onDisableSafeBrowsing and onClose', () => {
        const props = { ...defaultProps, entry: makeEntry({ reason: 'FilteredSafeBrowsing' }) };
        render(() => <ActionFooter {...props} />);
        fireEvent.click(screen.getByTestId('query-log-detail-action-disable-browsing-security'));
        expect(props.onDisableSafeBrowsing).toHaveBeenCalled();
        expect(props.onClose).toHaveBeenCalled();
    });

    it('parental control → Disable parental calls onDisableParental and onClose', () => {
        const props = { ...defaultProps, entry: makeEntry({ reason: 'FilteredParental' }) };
        render(() => <ActionFooter {...props} />);
        fireEvent.click(screen.getByTestId('query-log-detail-action-disable-parental'));
        expect(props.onDisableParental).toHaveBeenCalled();
        expect(props.onClose).toHaveBeenCalled();
    });

    it('safe search → Disable safe search calls onDisableSafeSearch and onClose', () => {
        const props = { ...defaultProps, entry: makeEntry({ reason: 'FilteredSafeSearch' }) };
        render(() => <ActionFooter {...props} />);
        fireEvent.click(screen.getByTestId('query-log-detail-action-disable-safe-search'));
        expect(props.onDisableSafeSearch).toHaveBeenCalled();
        expect(props.onClose).toHaveBeenCalled();
    });

    it('dns rewrite → Remove + Edit buttons; clicking Edit passes the matched rewrite rule', () => {
        const rewrite: RewriteEntry = { domain: 'example.org', answer: '192.0.2.1' };
        vi.mocked(findRewriteRuleByDomain).mockReturnValue(rewrite);
        const props = { ...defaultProps, entry: makeEntry({ reason: 'RewriteRule' }) };
        render(() => <ActionFooter {...props} />);
        expect(
            screen.getByTestId('query-log-detail-action-remove-dns-rewrite'),
        ).toBeInTheDocument();
        fireEvent.click(screen.getByTestId('query-log-detail-action-edit-dns-rewrite'));
        expect(props.onEditRewrite).toHaveBeenCalledWith(rewrite);
        expect(props.onClose).toHaveBeenCalled();
    });

    it('dns rewrite → footer hidden when no rewrite rule matches', () => {
        render(() => (
            <ActionFooter {...defaultProps} entry={makeEntry({ reason: 'RewriteRule' })} />
        ));
        expect(screen.queryByTestId('query-log-detail-action-footer')).not.toBeInTheDocument();
    });

    it('error entry → footer hidden entirely', () => {
        render(() => (
            <ActionFooter {...defaultProps} entry={makeEntry({ reason: 'FilteredInvalid' })} />
        ));
        expect(screen.queryByTestId('query-log-detail-action-footer')).not.toBeInTheDocument();
    });
});
