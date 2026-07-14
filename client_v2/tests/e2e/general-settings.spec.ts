import { test, expect, type Locator } from '@playwright/test';
import { execFileSync } from 'child_process';

import { login } from '../helpers/login';

const DNS_HOST = '127.0.0.1';
const DNS_PORT = 5353;

const lookupDomain = (domain: string): string =>
    execFileSync('nslookup', [`-port=${DNS_PORT}`, domain, DNS_HOST], {
        encoding: 'utf8',
        timeout: 15_000,
    });

const setToggleState = async (input: Locator, label: Locator, checked: boolean) => {
    if ((await input.isChecked()) === checked) {
        return;
    }

    await label.click();

    if (checked) {
        await expect(input).toBeChecked();
    } else {
        await expect(input).not.toBeChecked();
    }
};

const expectLookupToChange = async (domain: string, previousResult: string) => {
    await expect
        .poll(() => lookupDomain(domain), {
            timeout: 15_000,
            message: `Expected DNS lookup for ${domain} to change`,
        })
        .not.toBe(previousResult);
};

const expectLookupToMatch = async (domain: string, expectedResult: string) => {
    await expect
        .poll(() => lookupDomain(domain), {
            timeout: 15_000,
            message: `Expected DNS lookup for ${domain} to be restored`,
        })
        .toBe(expectedResult);
};

test.describe('General Settings', () => {
    // TODO(ik): fragile tests, need to rewrite later
    test.skip(() => !!process.env.CI, 'Skipped on CI: fragile tests');

    test.beforeEach(async ({ page }) => {
        await login(page);
    });

    test('should toggle browsing security feature and verify DNS changes', async ({ page }) => {
        await page.goto('/#settings');

        const browsingSecurity = page.locator('#safebrowsing');
        const browsingSecurityLabel = browsingSecurity.locator('xpath=following-sibling::*[1]');

        const initialState = await browsingSecurity.isChecked();
        const initialResult = lookupDomain('totalvirus.com');

        await setToggleState(browsingSecurity, browsingSecurityLabel, !initialState);
        await expectLookupToChange('totalvirus.com', initialResult);

        const toggledResult = lookupDomain('totalvirus.com');
        expect(toggledResult).not.toBe(initialResult);

        await setToggleState(browsingSecurity, browsingSecurityLabel, initialState);
        await expectLookupToMatch('totalvirus.com', initialResult);
    });

    test('should toggle parental control feature and verify DNS changes', async ({ page }) => {
        await page.goto('/#settings');

        const parentalControl = page.locator('#parental');
        const parentalControlLabel = parentalControl.locator('xpath=following-sibling::*[1]');

        const initialState = await parentalControl.isChecked();
        const initialResult = lookupDomain('pornhub.com');

        await setToggleState(parentalControl, parentalControlLabel, !initialState);
        await expectLookupToChange('pornhub.com', initialResult);

        const toggledResult = lookupDomain('pornhub.com');
        expect(toggledResult).not.toBe(initialResult);

        await setToggleState(parentalControl, parentalControlLabel, initialState);
        await expectLookupToMatch('pornhub.com', initialResult);
    });

    test('should toggle safe search feature', async ({ page }) => {
        await page.goto('/#settings');

        const safeSearch = page.locator('#safesearch');
        const safeSearchLabel = safeSearch.locator('xpath=following-sibling::*[1]');

        const initialState = await safeSearch.isChecked();

        await setToggleState(safeSearch, safeSearchLabel, !initialState);
        await setToggleState(safeSearch, safeSearchLabel, initialState);
    });
});
