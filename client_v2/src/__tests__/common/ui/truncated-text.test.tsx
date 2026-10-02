import { render, screen } from '@solidjs/testing-library';
import { describe, it, expect, beforeEach, vi } from 'vitest';

import { TruncatedText } from 'panel/common/ui/TruncatedText';
import { mockMatchMedia } from 'panel/__tests__/helpers/matchMedia';

/** jsdom reports every element as 0×0, so the measurement is stubbed. */
const truncated = vi.hoisted(() => ({ value: false }));

vi.mock('panel/hooks/useIsTruncated', () => ({
    useIsTruncated: () => () => truncated.value,
}));

const LONG_TEXT = 'very-long-subdomain.tracking-vendor.example.com';

describe('TruncatedText', () => {
    beforeEach(() => {
        truncated.value = false;
        mockMatchMedia(false);
    });

    it('renders the text without a native title', () => {
        const { container } = render(() => <TruncatedText text={LONG_TEXT} testId="cell" />);

        expect(screen.getByTestId('cell')).toHaveTextContent(LONG_TEXT);
        expect(container.querySelector('[title]')).toBeNull();
    });

    it('renders no tooltip while the text fits', () => {
        const { container } = render(() => <TruncatedText text={LONG_TEXT} />);

        expect(container.querySelector('[data-scope="tooltip"]')).toBeNull();
    });

    it('reveals the full text in a tooltip once it is clipped', () => {
        truncated.value = true;

        const { container } = render(() => <TruncatedText text={LONG_TEXT} />);

        expect(container.querySelector('[data-part="content"]')).toHaveTextContent(LONG_TEXT);
        expect(container.querySelector('[title]')).toBeNull();
    });

    it('renders no tooltip on touch devices', () => {
        truncated.value = true;
        mockMatchMedia((query) => query === '(hover: none)');

        const { container } = render(() => <TruncatedText text={LONG_TEXT} />);

        expect(screen.getByText(LONG_TEXT)).toBeInTheDocument();
        expect(container.querySelector('[data-scope="tooltip"]')).toBeNull();
    });
});
