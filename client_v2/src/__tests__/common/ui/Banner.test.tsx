import { render, screen } from '@solidjs/testing-library';
import { describe, it, expect, vi } from 'vitest';
import userEvent from '@testing-library/user-event';

import { Banner } from 'panel/common/ui/Banner';

describe('Banner', () => {
    it('renders the message slot', () => {
        render(() => <Banner variant="info" message="Test message" />);
        expect(screen.getByText('Test message')).toBeInTheDocument();
    });

    it('renders the action slot when provided', () => {
        render(() => (
            <Banner
                variant="info"
                message="Test"
                action={<button type="button">Click me</button>}
            />
        ));
        expect(screen.getByRole('button', { name: 'Click me' })).toBeInTheDocument();
    });

    it('applies the correct variant class', () => {
        const { container } = render(() => <Banner variant="warning" message="Warning!" />);
        const banner = container.firstElementChild as HTMLElement;
        expect(banner.className).toContain('warning');
    });

    it('sets role="alert" and aria-live="assertive" for critical variant', () => {
        const { container } = render(() => <Banner variant="critical" message="Critical!" />);
        const banner = container.firstElementChild as HTMLElement;
        expect(banner.getAttribute('role')).toBe('alert');
        expect(banner.getAttribute('aria-live')).toBe('assertive');
    });

    it('sets role="status" and aria-live="polite" for non-critical variants', () => {
        const { container } = render(() => <Banner variant="info" message="Info" />);
        const banner = container.firstElementChild as HTMLElement;
        expect(banner.getAttribute('role')).toBe('status');
        expect(banner.getAttribute('aria-live')).toBe('polite');
    });

    it('forwards the data-testid prop', () => {
        render(() => <Banner variant="info" message="Test" data-testid="custom-banner" />);
        expect(screen.getByTestId('custom-banner')).toBeInTheDocument();
    });

    it('renders the close button when onClose is provided', () => {
        const onClose = vi.fn();
        render(() => (
            <Banner variant="info" message="Test" onClose={onClose} data-testid="test-banner" />
        ));
        const closeButton = screen.getByTestId('test-banner-close');
        expect(closeButton).toBeInTheDocument();
        expect(closeButton.getAttribute('aria-label')).toBe('Close notification');
    });

    it('does not render the close button when onClose is not provided', () => {
        render(() => <Banner variant="info" message="Test" data-testid="test-banner" />);
        expect(screen.queryByTestId('test-banner-close')).not.toBeInTheDocument();
    });

    it('calls onClose when the close button is clicked', async () => {
        const user = userEvent.setup();
        const onClose = vi.fn();
        render(() => (
            <Banner variant="info" message="Test" onClose={onClose} data-testid="test-banner" />
        ));
        const closeButton = screen.getByTestId('test-banner-close');
        await user.click(closeButton);
        expect(onClose).toHaveBeenCalledTimes(1);
    });
});
