import { describe, expect, it } from 'vitest';
import {
    getDetailModalActions,
    type DetailModalActionContext,
} from 'panel/components/QueryLog/blocks/DetailModal/actions';

const ctx = (
    overrides: Partial<DetailModalActionContext> = {},
): DetailModalActionContext => ({
    hasServiceId: false,
    canDisableFilter: false,
    hasRewriteRule: false,
    ...overrides,
});

describe('getDetailModalActions', () => {
    it('allowed (allowlists) → Block', () => {
        expect(getDetailModalActions('allowlists', ctx())).toEqual(['block']);
    });

    it('processed (none) → Add to allowlist + Block, in order', () => {
        expect(getDetailModalActions('none', ctx())).toEqual([
            'add-to-allowlist',
            'block',
        ]);
    });

    it('blocked by custom rules → Add to allowlist only', () => {
        expect(getDetailModalActions('custom_filtering_rules', ctx())).toEqual([
            'add-to-allowlist',
        ]);
    });

    it('blocked by filter → Add to allowlist + Disable filter when filter found', () => {
        expect(
            getDetailModalActions(
                'blocked_by_filter',
                ctx({ canDisableFilter: true }),
            ),
        ).toEqual(['add-to-allowlist', 'disable-filter']);
    });

    it('blocked by filter → Add to allowlist only when filter not found', () => {
        expect(getDetailModalActions('blocked_by_filter', ctx())).toEqual([
            'add-to-allowlist',
        ]);
    });

    it('blocked service → Add to allowlist + Allow service when service id present', () => {
        expect(
            getDetailModalActions('blocked_services', ctx({ hasServiceId: true })),
        ).toEqual(['add-to-allowlist', 'allow-service']);
    });

    it('blocked service → Add to allowlist only when service id missing', () => {
        expect(getDetailModalActions('blocked_services', ctx())).toEqual([
            'add-to-allowlist',
        ]);
    });

    it('blocked threats → Add to allowlist + Disable Browsing security', () => {
        expect(getDetailModalActions('blocked_threats', ctx())).toEqual([
            'add-to-allowlist',
            'disable-browsing-security',
        ]);
    });

    it('parental → Add to allowlist + Disable Parental control', () => {
        expect(getDetailModalActions('blocked_by_parental_control', ctx())).toEqual(
            ['add-to-allowlist', 'disable-parental'],
        );
    });

    it('safe search → Add to allowlist + Disable Safe search', () => {
        expect(getDetailModalActions('safe_search', ctx())).toEqual([
            'add-to-allowlist',
            'disable-safe-search',
        ]);
    });

    it('dns rewrites → Remove + Edit DNS rewrite when rule found', () => {
        expect(
            getDetailModalActions('dns_rewrites', ctx({ hasRewriteRule: true })),
        ).toEqual(['remove-dns-rewrite', 'edit-dns-rewrite']);
    });

    it('dns rewrites → no actions when rule not found', () => {
        expect(getDetailModalActions('dns_rewrites', ctx())).toEqual([]);
    });

    it('error → no actions', () => {
        expect(getDetailModalActions('error', ctx())).toEqual([]);
    });
});
