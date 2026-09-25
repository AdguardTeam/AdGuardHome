import { describe, it, expect, vi } from 'vitest';
import { render, screen, cleanup, waitFor } from '@solidjs/testing-library';

import { useIsDarkTheme } from 'panel/hooks/useTheme';

const Harness = () => {
    const isDark = useIsDarkTheme();

    return <div data-testid="harness" data-dark={String(isDark())} />;
};

/** Applies a theme the way `setUITheme` does, through the `<html>` attribute. */
const applyTheme = (theme: 'light' | 'dark') => {
    document.documentElement.dataset.theme = theme;
};

const renderHarness = () => {
    render(() => <Harness />);

    return screen.getByTestId('harness');
};

describe('useIsDarkTheme', () => {
    it('reports false while no theme is applied', () => {
        expect(renderHarness()).toHaveAttribute('data-dark', 'false');
    });

    it('reports false for the light theme', () => {
        applyTheme('light');

        expect(renderHarness()).toHaveAttribute('data-dark', 'false');
    });

    it('reports true for the dark theme', () => {
        applyTheme('dark');

        expect(renderHarness()).toHaveAttribute('data-dark', 'true');
    });

    it('re-evaluates when the theme changes', async () => {
        applyTheme('light');
        const harness = renderHarness();

        expect(harness).toHaveAttribute('data-dark', 'false');

        applyTheme('dark');

        // MutationObserver callbacks are queued, so the update is not synchronous.
        await waitFor(() => expect(harness).toHaveAttribute('data-dark', 'true'));
    });

    it('stops observing the theme attribute on unmount', () => {
        const disconnect = vi.spyOn(MutationObserver.prototype, 'disconnect');
        renderHarness();

        cleanup();

        expect(disconnect).toHaveBeenCalled();
        disconnect.mockRestore();
    });
});
