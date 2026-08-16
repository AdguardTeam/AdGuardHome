import { render, screen, waitFor } from '@solidjs/testing-library';
import { describe, it, expect, vi } from 'vitest';
import userEvent from '@testing-library/user-event';

import { FileBrowseButton } from 'panel/components/Encryption/blocks/AddTlsCert/FileBrowseButton';

describe('FileBrowseButton', () => {
    it('renders a browse button', () => {
        const onFileSelect = vi.fn();
        render(() => <FileBrowseButton onFileSelect={onFileSelect} />);
        expect(screen.getByRole('button')).toBeInTheDocument();
    });

    it('renders a hidden file input', () => {
        const onFileSelect = vi.fn();
        const { container } = render(() => <FileBrowseButton onFileSelect={onFileSelect} />);
        const fileInput = container.querySelector('input[type="file"]');
        expect(fileInput).toBeInTheDocument();
        expect(fileInput?.getAttribute('aria-hidden')).toBe('true');
    });

    it('calls onFileSelect with file content when a file is selected', async () => {
        const user = userEvent.setup();
        const onFileSelect = vi.fn();
        const { container } = render(() => <FileBrowseButton onFileSelect={onFileSelect} />);

        const fileInput = container.querySelector('input[type="file"]') as HTMLInputElement;
        const file = new File(
            ['-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----'],
            'cert.pem',
            { type: 'application/x-pem-file' },
        );

        await user.upload(fileInput, file);

        await waitFor(() => {
            expect(onFileSelect).toHaveBeenCalledOnce();
        });
        expect(onFileSelect).toHaveBeenCalledWith(
            '-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----',
        );
    });

    it('does not call onFileSelect when no file is selected', async () => {
        const user = userEvent.setup();
        const onFileSelect = vi.fn();
        const { container } = render(() => <FileBrowseButton onFileSelect={onFileSelect} />);

        const fileInput = container.querySelector('input[type="file"]') as HTMLInputElement;
        await user.upload(fileInput, []);

        expect(onFileSelect).not.toHaveBeenCalled();
    });
});
