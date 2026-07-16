import { describe, it, expect, afterEach } from 'vitest';
import { render, screen, cleanup } from '@solidjs/testing-library';
import { Tooltip } from 'panel/common/ui/Tooltip';

describe('Tooltip', () => {
    afterEach(() => cleanup());

    it('renders trigger and content elements', () => {
        render(() => (
            <Tooltip content={<div data-testid="tooltip-content">Info</div>}>
                <div data-testid="trigger">Hover me</div>
            </Tooltip>
        ));

        expect(screen.getByTestId('trigger')).toBeTruthy();
        // Content is rendered but hidden (Zag.js sets hidden attribute)
        const content = screen.queryByTestId('tooltip-content');
        expect(content).toBeTruthy();
    });

    it('does not render tooltip content when disabled', () => {
        render(() => (
            <Tooltip disabled content={<div data-testid="tooltip-content">Info</div>}>
                <div data-testid="trigger">Hover me</div>
            </Tooltip>
        ));

        // When disabled, the component short-circuits and renders children only
        expect(screen.getByTestId('trigger')).toBeTruthy();
        expect(screen.queryByTestId('tooltip-content')).toBeNull();
    });

    it('renders with position prop without errors', () => {
        const positions = ['top', 'bottomLeft', 'bottomRight', 'topLeft', 'topRight'] as const;
        for (const pos of positions) {
            render(() => (
                <Tooltip position={pos} content={<div>Info</div>}>
                    <div>Trigger</div>
                </Tooltip>
            ));
            cleanup();
        }
        // No errors thrown — all positions valid
        expect(true).toBe(true);
    });

    it('renders without position prop (default placement)', () => {
        render(() => (
            <Tooltip content={<div data-testid="tooltip-content">Info</div>}>
                <div data-testid="trigger">Hover me</div>
            </Tooltip>
        ));

        expect(screen.getByTestId('trigger')).toBeTruthy();
        expect(screen.getByTestId('tooltip-content')).toBeTruthy();
    });

    it('applies overlayClass to tooltip content wrapper', () => {
        render(() => (
            <Tooltip
                content={<div data-testid="tooltip-content">Info</div>}
                overlayClass="custom-overlay"
            >
                <div>Trigger</div>
            </Tooltip>
        ));

        // overlayClass is applied to ArkTooltip.Content wrapper (parent of content)
        const content = screen.getByTestId('tooltip-content');
        const wrapper = content.parentElement;
        expect(wrapper).toBeTruthy();
        expect(wrapper!.className).toContain('custom-overlay');
    });

    it('applies class to wrapper', () => {
        render(() => (
            <Tooltip content={<div>Info</div>} class="custom-wrapper">
                <div data-testid="child">Trigger</div>
            </Tooltip>
        ));

        const child = screen.getByTestId('child');
        expect(child.parentElement?.className).toContain('custom-wrapper');
    });

    // Note: openDelay/closeDelay, multi-instance coordination, and fast-hover
    // behavior are verified through manual testing (see plan.md Task 5).
    // Ark UI Tooltip's internal state machine uses setTimeout and
    // queueMicrotask which makes jsdom-based timing tests unreliable.
});
