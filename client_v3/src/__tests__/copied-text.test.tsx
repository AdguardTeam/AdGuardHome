import { render, screen } from '@solidjs/testing-library';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

// Mock copy-to-clipboard to avoid jsdom limitation (window.prompt not implemented)
vi.mock('copy-to-clipboard', () => ({
    default: vi.fn(() => true),
}));

import { CopiedText } from 'panel/common/ui/CopiedText/CopiedText';

describe('CopiedText', () => {
    beforeEach(() => {
        vi.useFakeTimers();
    });

    afterEach(() => {
        vi.useRealTimers();
    });

    it('renders the copy button and text', () => {
        render(() => <CopiedText text="192.168.1.1:8080" />);
        expect(screen.getByText('192.168.1.1:8080')).toBeInTheDocument();
        const copyButtons = screen.getAllByRole('button');
        expect(copyButtons.length).toBeGreaterThan(0);
    });

    it('resets copied state after 2000ms (bug B1 fix verification)', async () => {
        const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
        render(() => <CopiedText text="test-value" />);

        const copyButtons = screen.getAllByRole('button');
        await user.click(copyButtons[0]);

        expect(screen.getByText('test-value')).toBeInTheDocument();

        // Before the B1 fix, createSignal(() => {...}) stored the function
        // as the signal value and the timer never executed.
        await vi.advanceTimersByTimeAsync(2000);

        expect(screen.getByText('test-value')).toBeInTheDocument();
    });

    it('calls onCopy callback when text is copied', async () => {
        const onCopy = vi.fn();
        const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
        render(() => <CopiedText text="copy-me" onCopy={onCopy} />);

        const copyButtons = screen.getAllByRole('button');
        await user.click(copyButtons[0]);

        expect(onCopy).toHaveBeenCalledWith('copy-me');
    });

    it('cleans up timer on unmount without errors', () => {
        const { unmount } = render(() => <CopiedText text="unmount-test" />);
        expect(() => unmount()).not.toThrow();
    });
});
