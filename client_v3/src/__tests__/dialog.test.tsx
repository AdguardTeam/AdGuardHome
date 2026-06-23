import { render, screen } from '@solidjs/testing-library';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi } from 'vitest';
import { Dialog } from 'panel/common/ui/Dialog/Dialog';

describe('Dialog', () => {
    it('renders when visible', () => {
        render(() => (
            <Dialog visible onClose={() => {}} title="Test Dialog">
                <p>Body content</p>
            </Dialog>
        ));
        expect(screen.getByText('Test Dialog')).toBeInTheDocument();
        expect(screen.getByText('Body content')).toBeInTheDocument();
    });

    it('does not render when not visible', () => {
        render(() => (
            <Dialog visible={false} onClose={() => {}} title="Hidden">
                <p>Should not appear</p>
            </Dialog>
        ));
        // Ark UI uses Presence — content is in DOM but hidden
        const title = screen.queryByText('Hidden');
        expect(title?.closest('[data-state="closed"]')).toBeTruthy();
    });

    it('calls onClose when close button clicked', async () => {
        const onClose = vi.fn();
        render(() => <Dialog visible onClose={onClose} title="Closable" />);
        const closeButton = screen.getByRole('button');
        await userEvent.click(closeButton);
        expect(onClose).toHaveBeenCalled();
    });

    it('renders without mask when mask=false', () => {
        render(() => <Dialog visible mask={false} onClose={() => {}} title="No Mask" />);
        expect(document.querySelector('[data-part="backdrop"]')).not.toBeInTheDocument();
    });

    it('renders mask by default', () => {
        render(() => <Dialog visible onClose={() => {}} title="With Mask" />);
        expect(document.querySelector('[data-part="backdrop"]')).toBeInTheDocument();
    });
});
