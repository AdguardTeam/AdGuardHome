import { describe, it, expect, beforeAll } from 'vitest';
import { render, screen } from '@solidjs/testing-library';

import { QueriesTooltip } from 'panel/common/ui/QueriesTooltip';

// jsdom lacks ResizeObserver, which floating-ui (used by Zag positioning) needs.
beforeAll(() => {
    if (!global.ResizeObserver) {
        global.ResizeObserver = class {
            observe() {}
            unobserve() {}
            disconnect() {}
        };
    }
});

describe('QueriesTooltip', () => {
    it('renders tooltip when count is above the threshold', () => {
        const { container } = render(() => (
            <QueriesTooltip count={1001}>
                <span>Trigger</span>
            </QueriesTooltip>
        ));
        expect(screen.getByText('Trigger')).toBeInTheDocument();
        expect(container.querySelector('[data-scope="tooltip"]')).not.toBeNull();
        expect(container.textContent).toContain('queries');
    });

    it('renders without tooltip when count equals the threshold', () => {
        const { container } = render(() => (
            <QueriesTooltip count={1000}>
                <span>Trigger</span>
            </QueriesTooltip>
        ));
        expect(screen.getByText('Trigger')).toBeInTheDocument();
        expect(container.querySelector('[data-scope="tooltip"]')).toBeNull();
    });

    it('renders without tooltip when count is below the threshold', () => {
        const { container } = render(() => (
            <QueriesTooltip count={0}>
                <span>Trigger</span>
            </QueriesTooltip>
        ));
        expect(screen.getByText('Trigger')).toBeInTheDocument();
        expect(container.querySelector('[data-scope="tooltip"]')).toBeNull();
    });
});
