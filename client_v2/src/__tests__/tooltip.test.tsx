import { describe, it, expect, vi, beforeEach, afterEach, beforeAll } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@solidjs/testing-library';

import { Tooltip } from 'panel/common/ui/Tooltip';

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

// Restore real matchMedia between test suites.  The global mock from
// src/__tests__/setup.ts returns matches: false for every query (desktop),
// which is what most tests need.  Touch tests in this file replace it
// temporarily so that (hover: none) returns true.
const realMatchMedia = window.matchMedia;

function mockTouchDevice() {
    window.matchMedia = (query: string): MediaQueryList => {
        const matches = query === '(hover: none)';
        return {
            matches,
            media: query,
            onchange: null,
            addListener: () => {},
            removeListener: () => {},
            addEventListener: () => {},
            removeEventListener: () => {},
            dispatchEvent: () => false,
        } as MediaQueryList;
    };
}

function mockDesktop() {
    window.matchMedia = realMatchMedia;
}

describe('Tooltip', () => {
    beforeEach(() => {
        mockDesktop();
    });

    afterEach(() => {
        // Clean up any remaining fake timers between tests.
        vi.useRealTimers();
    });

    // ---- general rendering ----

    it('renders children', () => {
        render(() => (
            <Tooltip content="Help text">
                <span>Trigger</span>
            </Tooltip>
        ));
        expect(screen.getByText('Trigger')).toBeInTheDocument();
    });

    it('renders content element (hidden when closed)', () => {
        render(() => (
            <Tooltip content="Help text">
                <span>Trigger</span>
            </Tooltip>
        ));
        const content = screen.getByText('Help text');
        expect(content).toBeInTheDocument();
        expect(content.closest('[hidden]')).toBeTruthy();
    });

    it('renders without tooltip wrapper when disabled', () => {
        const { container } = render(() => (
            <Tooltip content="Help text" disabled>
                <span>Trigger</span>
            </Tooltip>
        ));
        expect(screen.getByText('Trigger')).toBeInTheDocument();
        expect(container.querySelector('[data-scope="tooltip"]')).toBeNull();
    });

    it('applies custom class to the trigger wrapper', () => {
        const { container } = render(() => (
            <Tooltip content="Help text" class="my-custom-class">
                <span>Trigger</span>
            </Tooltip>
        ));
        const trigger = container.querySelector('[data-part="trigger"]');
        expect(trigger?.className).toContain('my-custom-class');
    });

    it('applies overlayClass to the content', () => {
        render(() => (
            <Tooltip content="Help text" overlayClass="custom-overlay">
                <span>Trigger</span>
            </Tooltip>
        ));
        const content = screen.getByText('Help text').closest('[data-part="content"]');
        expect(content?.className).toContain('custom-overlay');
    });

    // ---- desktop hover (default matchMedia mock → (hover: none) = false) ----

    it('opens on hover after open delay on desktop', async () => {
        render(() => (
            <Tooltip content="Tooltip text">
                <span>Hover me</span>
            </Tooltip>
        ));

        const triggerEl = screen.getByText('Hover me').closest('[data-part="trigger"]')!;
        const content = screen.getByText('Tooltip text').closest('[data-part="content"]')!;

        expect(content.hasAttribute('hidden')).toBe(true);

        fireEvent.pointerOver(triggerEl, { pointerType: 'mouse' });

        // SHOW_DELAY is 200 ms; use generous timeout (real timers, ~200 ms actual).
        await waitFor(
            () => {
                expect(content.hasAttribute('hidden')).toBe(false);
            },
            { timeout: 1000 },
        );
    });

    it('closes on pointer leave after close delay on desktop', async () => {
        render(() => (
            <Tooltip content="Tooltip text">
                <span>Hover me</span>
            </Tooltip>
        ));

        const triggerEl = screen.getByText('Hover me').closest('[data-part="trigger"]')!;
        const content = screen.getByText('Tooltip text').closest('[data-part="content"]')!;

        // Open via hover.
        fireEvent.pointerOver(triggerEl, { pointerType: 'mouse' });
        await waitFor(
            () => {
                expect(content.hasAttribute('hidden')).toBe(false);
            },
            { timeout: 1000 },
        );

        // Leave → closes after HIDE_DELAY (300 ms).
        // pointerleave does NOT bubble, so fire on the trigger element itself.
        fireEvent.pointerLeave(triggerEl);
        await waitFor(
            () => {
                expect(content.hasAttribute('hidden')).toBe(true);
            },
            { timeout: 1000 },
        );
    });

    // ---- touch (override matchMedia to report (hover: none) = true) ----

    describe('touch device', () => {
        beforeEach(() => {
            mockTouchDevice();
        });

        it('opens on click (tap)', async () => {
            render(() => (
                <Tooltip content="Touch help">
                    <span>Tap me</span>
                </Tooltip>
            ));

            const trigger = screen.getByText('Tap me');
            const content = screen.getByText('Touch help').closest('[data-part="content"]')!;

            expect(content.hasAttribute('hidden')).toBe(true);

            fireEvent.click(trigger);

            await waitFor(() => {
                expect(content.hasAttribute('hidden')).toBe(false);
            });
        });

        it('does not close on second click (close is outside-tap only)', async () => {
            render(() => (
                <Tooltip content="Touch help">
                    <span>Tap me</span>
                </Tooltip>
            ));

            const trigger = screen.getByText('Tap me');
            const content = screen.getByText('Touch help').closest('[data-part="content"]')!;

            // Open.
            fireEvent.click(trigger);
            await waitFor(() => {
                expect(content.hasAttribute('hidden')).toBe(false);
            });

            // Second tap on trigger — stays open (no toggle).
            fireEvent.click(trigger);
            expect(content.hasAttribute('hidden')).toBe(false);

            // Outside tap closes.
            fireEvent.pointerDown(document.body);
            await waitFor(() => {
                expect(content.hasAttribute('hidden')).toBe(true);
            });
        });

        it('closes when tapping outside (document pointerdown)', async () => {
            render(() => (
                <Tooltip content="Touch help">
                    <span>Tap me</span>
                </Tooltip>
            ));

            const trigger = screen.getByText('Tap me');
            const content = screen.getByText('Touch help').closest('[data-part="content"]')!;

            // Open.
            fireEvent.click(trigger);
            await waitFor(() => {
                expect(content.hasAttribute('hidden')).toBe(false);
            });

            // Tap on document.body (outside both trigger and content).
            fireEvent.pointerDown(document.body);
            await waitFor(() => {
                expect(content.hasAttribute('hidden')).toBe(true);
            });
        });

        it('does NOT close when tapping inside open content (interactive)', async () => {
            render(() => (
                <Tooltip content={<a href="#">Link inside</a>}>
                    <span>Tap me</span>
                </Tooltip>
            ));

            const trigger = screen.getByText('Tap me');

            // Open via tap.
            fireEvent.click(trigger);

            const contentEl = screen.getByText('Link inside').closest('[data-part="content"]')!;
            await waitFor(
                () => {
                    expect(contentEl.hasAttribute('hidden')).toBe(false);
                },
                { timeout: 1000 },
            );

            // Tap on the link inside the open content.
            fireEvent.pointerDown(screen.getByText('Link inside'));
            // Should stay open.
            expect(contentEl.hasAttribute('hidden')).toBe(false);
        });
    });
});
