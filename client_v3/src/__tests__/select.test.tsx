import { render } from '@solidjs/testing-library';
import { describe, it, expect } from 'vitest';
import { Select } from '../common/controls/Select/Select';

const OPTIONS = [
    { value: '0.0.0.0', label: 'All interfaces' },
    { value: '127.0.0.1', label: 'Loopback' },
];

describe('Select', () => {
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
