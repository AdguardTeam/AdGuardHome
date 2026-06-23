import { render } from '@solidjs/testing-library';
import userEvent from '@testing-library/user-event';
import { describe, it, expect } from 'vitest';
import { Select } from '../common/controls/Select/Select';

const OPTIONS = [
    { value: '0.0.0.0', label: 'All interfaces' },
    { value: '127.0.0.1', label: 'Loopback' },
];

describe('Select', () => {
    it('opens the menu when clicking anywhere on the control (not just text)', async () => {
        const user = userEvent.setup();
        render(() => (
            <Select
                options={OPTIONS}
                value={OPTIONS[0]}
                onChange={() => {}}
                placeholder="Select interface"
                id="test-select"
            />
        ));

        // The trigger button should contain both the value text and the indicator.
        const trigger = document.querySelector(
            '[data-scope="select"][data-part="trigger"]',
        ) as HTMLElement;

        expect(trigger).toBeTruthy();

        // Click on the trigger (the whole button area).
        await user.click(trigger);

        // Menu content should appear.
        const content = document.querySelector('[data-scope="select"][data-part="content"]');
        expect(content).toBeVisible();
    });

    it('renders the indicator inside the trigger button', () => {
        render(() => (
            <Select
                options={OPTIONS}
                value={OPTIONS[0]}
                onChange={() => {}}
                placeholder="Select interface"
            />
        ));

        const trigger = document.querySelector(
            '[data-scope="select"][data-part="trigger"]',
        ) as HTMLElement;
        const indicator = trigger?.querySelector('[data-part="indicator"]');

        // The indicator (arrow icon) should be inside the trigger.
        expect(indicator).toBeTruthy();
    });
});
