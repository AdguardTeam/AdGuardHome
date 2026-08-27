import { describe, test, expect } from 'vitest';

interface WebService {
    id: string;
    name: string;
    icon_svg: string;
    rules: string[];
}

const filterServices = (
    services: WebService[],
    search: string,
    groupFilter: string[],
    serviceGroupMap: Map<string, string>,
    blockedIds: Set<string>,
    showBlockedOnly: boolean,
    showUnblockedOnly: boolean,
): WebService[] => {
    let filtered = services;

    if (groupFilter.length > 0) {
        const selected = new Set(groupFilter);
        filtered = filtered.filter((s) => {
            const groupId = serviceGroupMap.get(s.id);
            return groupId && selected.has(groupId);
        });
    }

    if (showBlockedOnly) {
        filtered = filtered.filter((s) => blockedIds.has(s.id));
    } else if (showUnblockedOnly) {
        filtered = filtered.filter((s) => !blockedIds.has(s.id));
    }

    const term = search.trim().toLowerCase();
    if (term) {
        filtered = filtered.filter(
            (s) => s.name.toLowerCase().includes(term) || s.id.toLowerCase().includes(term),
        );
    }

    return filtered;
};

describe('filterServices', () => {
    const services: WebService[] = [
        { id: 'telegram', name: 'Telegram', icon_svg: '<svg/>', rules: [] },
        { id: 'whatsapp', name: 'WhatsApp', icon_svg: '<svg/>', rules: [] },
        { id: 'steam', name: 'Steam', icon_svg: '<svg/>', rules: [] },
        { id: 'epic_games', name: 'Epic Games', icon_svg: '<svg/>', rules: [] },
    ];

    const serviceGroupMap = new Map([
        ['telegram', 'messaging'],
        ['whatsapp', 'messaging'],
        ['steam', 'gaming'],
        ['epic_games', 'gaming'],
    ]);

    const blockedIds = new Set(['telegram', 'steam']);
    const noFilters = (showBlockedOnly = false, showUnblockedOnly = false): WebService[] =>
        filterServices(
            services,
            '',
            [],
            serviceGroupMap,
            blockedIds,
            showBlockedOnly,
            showUnblockedOnly,
        );

    test('no filters returns all services', () => {
        expect(noFilters()).toHaveLength(4);
    });

    test('search by name (case-insensitive)', () => {
        const result = filterServices(
            services,
            'tel',
            [],
            serviceGroupMap,
            blockedIds,
            false,
            false,
        );
        expect(result).toHaveLength(1);
        expect(result[0].id).toBe('telegram');
    });

    test('search by id', () => {
        const result = filterServices(
            services,
            'epic',
            [],
            serviceGroupMap,
            blockedIds,
            false,
            false,
        );
        expect(result).toHaveLength(1);
        expect(result[0].id).toBe('epic_games');
    });

    test('filter by group', () => {
        const result = filterServices(
            services,
            '',
            ['gaming'],
            serviceGroupMap,
            blockedIds,
            false,
            false,
        );
        expect(result).toHaveLength(2);
        expect(result.map((s) => s.id)).toEqual(['steam', 'epic_games']);
    });

    test('combined search + group filter', () => {
        const result = filterServices(
            services,
            'steam',
            ['gaming'],
            serviceGroupMap,
            blockedIds,
            false,
            false,
        );
        expect(result).toHaveLength(1);
        expect(result[0].id).toBe('steam');
    });

    test('no match returns empty array', () => {
        const result = filterServices(
            services,
            'xyz',
            [],
            serviceGroupMap,
            blockedIds,
            false,
            false,
        );
        expect(result).toHaveLength(0);
    });

    test('blocked only filter returns blocked services', () => {
        const result = noFilters(true);
        expect(result.map((s) => s.id)).toEqual(['telegram', 'steam']);
    });

    test('unblocked only filter returns unblocked services', () => {
        const result = noFilters(false, true);
        expect(result.map((s) => s.id)).toEqual(['whatsapp', 'epic_games']);
    });

    test('blocked only takes precedence when both state filters are active', () => {
        const result = noFilters(true, true);
        expect(result.map((s) => s.id)).toEqual(['telegram', 'steam']);
    });

    test('state filter combines with group filter', () => {
        const result = filterServices(
            services,
            '',
            ['gaming'],
            serviceGroupMap,
            blockedIds,
            false,
            true,
        );
        expect(result.map((s) => s.id)).toEqual(['epic_games']);
    });
});
