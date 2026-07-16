import { describe, it, expect, afterEach } from 'vitest';
import { render, screen, cleanup } from '@solidjs/testing-library';
import { Dropdown } from 'panel/common/ui/Dropdown';

describe('Dropdown', () => {
    afterEach(() => cleanup());

    it('renders trigger and menu content', () => {
        render(() => (
            <Dropdown noIcon menu={<div data-testid="menu-content">Menu</div>}>
                <div data-testid="click-trigger">Click me</div>
            </Dropdown>
        ));

        expect(screen.getByTestId('click-trigger')).toBeTruthy();
        // Content exists in DOM (Ark UI keeps it in DOM, hidden when closed)
        expect(screen.getByTestId('menu-content')).toBeTruthy();
    });

    it('renders with custom position', () => {
        render(() => (
            <Dropdown noIcon position="topRight" menu={<div>Menu</div>}>
                <div>Trigger</div>
            </Dropdown>
        ));

        // Should render without errors
        expect(screen.getByText('Trigger')).toBeTruthy();
        expect(screen.getByText('Menu')).toBeTruthy();
    });

    it('renders when disabled', () => {
        render(() => (
            <Dropdown noIcon disabled menu={<div>Menu</div>}>
                <div data-testid="trigger">Trigger</div>
            </Dropdown>
        ));

        expect(screen.getByTestId('trigger')).toBeTruthy();
    });

    // Note: Click-to-toggle behavior is verified through manual testing
    // (see plan.md Task 5). Ark UI Popover's controlled mode with
    // closeOnInteractOutside makes jsdom-based click testing unreliable
    // due to event delegation on portal-rendered content.
});
