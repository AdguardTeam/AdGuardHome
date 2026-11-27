import { describe, it, expect } from 'vitest';

import { SERVICES_BASE_LOCALE } from 'panel/common/intl/locales.generated';
import { SERVICE_GROUP_MESSAGES } from 'panel/common/intl/services.groups.generated';

/** Base-locale blocked-services group names, keyed by the API's group ID. */
const baseGroupNames: Record<string, string> = {};

for (const [key, value] of Object.entries(SERVICES_BASE_LOCALE)) {
    const match = /^servicesgroup\.(.+)\.name$/.exec(key);
    if (match) {
        baseGroupNames[match[1]] = typeof value === 'string' ? value : value.message;
    }
}

describe('services.groups.generated.ts', () => {
    it('maps every blocked-services group ID to a message', () => {
        expect(Object.keys(SERVICE_GROUP_MESSAGES).sort()).toEqual(
            Object.keys(baseGroupNames).sort(),
        );
    });

    it('resolves each group ID to the base-locale name', () => {
        for (const [id, name] of Object.entries(baseGroupNames)) {
            expect(SERVICE_GROUP_MESSAGES[id]?.()).toBe(name);
        }
    });
});
