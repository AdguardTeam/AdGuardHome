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
): WebService[] => {
    let filtered = services;

    if (groupFilter.length > 0) {
        const selected = new Set(groupFilter);
        filtered = filtered.filter((s) => {
            const groupId = serviceGroupMap.get(s.id);
            return groupId && selected.has(groupId);
        });
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

    test('no filters returns all services', () => {
        expect(filterServices(services, '', [], serviceGroupMap)).toHaveLength(4);
    });

    test('search by name (case-insensitive)', () => {
        const result = filterServices(services, 'tel', [], serviceGroupMap);
        expect(result).toHaveLength(1);
        expect(result[0].id).toBe('telegram');
    });

    test('search by id', () => {
        const result = filterServices(services, 'epic', [], serviceGroupMap);
        expect(result).toHaveLength(1);
        expect(result[0].id).toBe('epic_games');
    });

    test('filter by group', () => {
        const result = filterServices(services, '', ['gaming'], serviceGroupMap);
        expect(result).toHaveLength(2);
        expect(result.map((s) => s.id)).toEqual(['steam', 'epic_games']);
    });

    test('combined search + group filter', () => {
        const result = filterServices(services, 'steam', ['gaming'], serviceGroupMap);
        expect(result).toHaveLength(1);
        expect(result[0].id).toBe('steam');
    });

    test('no match returns empty array', () => {
        const result = filterServices(services, 'xyz', [], serviceGroupMap);
        expect(result).toHaveLength(0);
    });
});
