import { describe, it, expect } from 'vitest';
import { render, screen, cleanup } from '@solidjs/testing-library';

import { useIsDesktop, useIsMobile, useMediaQuery, useIsTouchDevice } from 'panel/hooks/useMediaQuery';
import { byViewportWidth, mockMatchMedia } from 'panel/__tests__/helpers/matchMedia';

const Harness = () => {
    const isDesktop = useIsDesktop();
    const isMobile = useIsMobile();
    const isTouch = useIsTouchDevice();
    const isWide = useMediaQuery('(min-width: 1200px)');

    return (
        <div
            data-testid="harness"
            data-desktop={String(isDesktop())}
            data-mobile={String(isMobile())}
            data-touch={String(isTouch())}
            data-wide={String(isWide())}
        />
    );
};

/** Renders the harness against a simulated `width`-pixel-wide viewport. */
const renderAt = (width: number) => {
    mockMatchMedia(byViewportWidth(width));
    render(() => <Harness />);

    return screen.getByTestId('harness');
};

describe('useMediaQuery', () => {
    it('reports the current match of the given query', () => {
        expect(renderAt(1300)).toHaveAttribute('data-wide', 'true');
    });

    it('reports false for a query that does not match', () => {
        expect(renderAt(900)).toHaveAttribute('data-wide', 'false');
    });

    it('re-evaluates when the media query result changes', () => {
        const media = mockMatchMedia(byViewportWidth(500));
        render(() => <Harness />);

        expect(screen.getByTestId('harness')).toHaveAttribute('data-desktop', 'false');

        media.set(byViewportWidth(1300));

        expect(screen.getByTestId('harness')).toHaveAttribute('data-wide', 'true');
    });

    it('unsubscribes from the media query on unmount', () => {
        const media = mockMatchMedia(byViewportWidth(1300));
        render(() => <Harness />);

        expect(media.listenerCount()).toBeGreaterThan(0);

        cleanup();

        expect(media.listenerCount()).toBe(0);
    });
});

describe('useIsDesktop', () => {
    it('reports false just below the 768px breakpoint', () => {
        expect(renderAt(767)).toHaveAttribute('data-desktop', 'false');
    });

    it('reports true from the 768px breakpoint up', () => {
        expect(renderAt(768)).toHaveAttribute('data-desktop', 'true');
    });
});

describe('useIsMobile', () => {
    it('reports true just below the 1024px breakpoint', () => {
        expect(renderAt(1023)).toHaveAttribute('data-mobile', 'true');
    });

    it('reports false from the 1024px breakpoint up', () => {
        expect(renderAt(1024)).toHaveAttribute('data-mobile', 'false');
    });

    it('follows the viewport when it crosses the breakpoint', () => {
        const media = mockMatchMedia(byViewportWidth(500));
        render(() => <Harness />);

        expect(screen.getByTestId('harness')).toHaveAttribute('data-mobile', 'true');

        media.set(byViewportWidth(1100));

        expect(screen.getByTestId('harness')).toHaveAttribute('data-mobile', 'false');
    });

    /**
     * `useIsMobile` and `useIsDesktop` deliberately use different breakpoints,
     * so mid-size viewports are reported by both hooks at once.  Every call site
     * relies on this, so pin it before "fixing" either hook.
     */
    it('overlaps with useIsDesktop between 768px and 1023px', () => {
        const harness = renderAt(900);

        expect(harness).toHaveAttribute('data-desktop', 'true');
        expect(harness).toHaveAttribute('data-mobile', 'true');
    });
});

describe('useIsTouchDevice', () => {
    it('reports true when the device cannot hover', () => {
        mockMatchMedia((query) => query === '(hover: none)');
        render(() => <Harness />);

        expect(screen.getByTestId('harness')).toHaveAttribute('data-touch', 'true');
    });

    it('reports false on devices with hover support', () => {
        expect(renderAt(1300)).toHaveAttribute('data-touch', 'false');
    });
});
