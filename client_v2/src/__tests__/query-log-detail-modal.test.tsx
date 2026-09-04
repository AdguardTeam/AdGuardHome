import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { ComponentProps } from 'solid-js';
import { render, screen, fireEvent } from '@solidjs/testing-library';
import intl from 'panel/common/intl';
import { DetailModal } from 'panel/components/QueryLog/blocks/DetailModal/DetailModal';
import type { NormalizedQueryLogItem } from 'panel/helpers/helpers';
import type { Filter } from 'panel/helpers/helpers';

const makeEntry = (
    overrides: Partial<NormalizedQueryLogItem> = {},
): NormalizedQueryLogItem => ({
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

const defaultProps: ComponentProps<typeof DetailModal> = {
    entry: makeEntry(),
    filters: [],
    services: [],
    whitelistFilters: [],
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

describe('DetailModal actions footer', () => {
    beforeEach(() => {
        vi.spyOn(intl, 'getMessage').mockImplementation((key) => key);
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    it('processed entry → Add to allowlist (primary) + Block (danger), in order', () => {
        render(() => (
            <DetailModal
                {...defaultProps}
                entry={makeEntry({ reason: 'NotFilteredNotFound' })}
            />
        ));
        const footer = screen.getByTestId('query-log-detail-action-footer');
        const buttons = footer.querySelectorAll('button');
        expect(buttons).toHaveLength(2);
        expect(buttons[0]).toHaveAttribute('data-action', 'allowlist');
        expect(buttons[1]).toHaveAttribute('data-action', 'block');
    });

    it('allowed entry → Block only', () => {
        render(() => (
            <DetailModal
                {...defaultProps}
                entry={makeEntry({ reason: 'NotFilteredWhiteList' })}
            />
        ));
        expect(
            screen.getByTestId('query-log-detail-action-block'),
        ).toBeInTheDocument();
        expect(
            screen.queryByTestId('query-log-detail-action-allowlist'),
        ).not.toBeInTheDocument();
    });

    it('blocked by filter → adds Disable filter; clicking calls onDisableFilter and closes', () => {
        const props = {
            ...defaultProps,
            entry: makeEntry({
                reason: 'FilteredBlackList',
                rules: [{ filter_list_id: 5 }],
            }),
            filters: [testFilter],
        };
        render(() => <DetailModal {...props} />);
        const btn = screen.getByTestId('query-log-detail-action-disable-filter');
        expect(btn).toHaveAttribute('data-action', 'disable-filter');
        fireEvent.click(btn);
        expect(props.onDisableFilter).toHaveBeenCalledWith(testFilter);
        expect(props.onClose).toHaveBeenCalled();
    });

    it('blocked service → Add to allowlist + Allow service', () => {
        render(() => (
            <DetailModal
                {...defaultProps}
                entry={makeEntry({
                    reason: 'FilteredBlockedService',
                    serviceName: 'amazon',
                })}
            />
        ));
        expect(
            screen.getByTestId('query-log-detail-action-allow-service'),
        ).toBeInTheDocument();
    });

    it('safe search → Add to allowlist + Disable Safe search', () => {
        render(() => (
            <DetailModal
                {...defaultProps}
                entry={makeEntry({ reason: 'FilteredSafeSearch' })}
            />
        ));
        expect(
            screen.getByTestId('query-log-detail-action-disable-safe-search'),
        ).toBeInTheDocument();
    });

    it('error entry → footer is hidden entirely', () => {
        render(() => (
            <DetailModal
                {...defaultProps}
                entry={makeEntry({ reason: 'FilteredInvalid' })}
            />
        ));
        expect(
            screen.queryByTestId('query-log-detail-action-footer'),
        ).not.toBeInTheDocument();
    });
});
