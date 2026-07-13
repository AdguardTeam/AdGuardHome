import { describe, expect, it } from 'vitest';
import { normalizeMac } from 'panel/helpers/form';

describe('normalizeMac', () => {
    it('normalizes bare hex (12 chars)', () => {
        expect(normalizeMac('AABBCCDDEEFF')).toBe('AA:BB:CC:DD:EE:FF');
    });

    it('normalizes bare hex lowercase', () => {
        expect(normalizeMac('aabbccddeeff')).toBe('AA:BB:CC:DD:EE:FF');
    });

    it('normalizes dash-separated', () => {
        expect(normalizeMac('AA-BB-CC-DD-EE-FF')).toBe('AA:BB:CC:DD:EE:FF');
    });

    it('normalizes dash-separated lowercase', () => {
        expect(normalizeMac('aa-bb-cc-dd-ee-ff')).toBe('AA:BB:CC:DD:EE:FF');
    });

    it('uppercases colon-separated', () => {
        expect(normalizeMac('aa:bb:cc:dd:ee:ff')).toBe('AA:BB:CC:DD:EE:FF');
    });

    it('returns already-normalized MACs unchanged', () => {
        expect(normalizeMac('AA:BB:CC:DD:EE:FF')).toBe('AA:BB:CC:DD:EE:FF');
    });

    it('converts dashes to colons (general dash handling)', () => {
        expect(normalizeMac('not-a-mac')).toBe('NOT:A:MAC');
    });

    it('handles empty string', () => {
        expect(normalizeMac('')).toBe('');
    });
});
