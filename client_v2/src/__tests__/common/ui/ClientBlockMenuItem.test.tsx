import { render, screen } from '@solidjs/testing-library';
import { describe, it, expect, vi } from 'vitest';
import userEvent from '@testing-library/user-event';

import { ClientBlockMenuItem } from 'panel/common/ui/ClientBlockConfirm';
import { copyInDom } from 'panel/__tests__/helpers/copy';

describe('ClientBlockMenuItem', () => {
    it('renders the block action as a button', async () => {
        const onClick = vi.fn();
        render(() => <ClientBlockMenuItem action="block" onClick={onClick} />);

        const button = screen.getByRole('button', { name: copyInDom('block_client') });
        expect(button.tagName).toBe('BUTTON');
        expect(button.getAttribute('type')).toBe('button');

        await userEvent.click(button);
        expect(onClick).toHaveBeenCalledWith('block');
    });

    it('renders the unblock action as a button', async () => {
        const onClick = vi.fn();
        render(() => <ClientBlockMenuItem action="unblock" onClick={onClick} />);

        const button = screen.getByRole('button', { name: copyInDom('unblock_client') });
        expect(button.tagName).toBe('BUTTON');
        expect(button.getAttribute('type')).toBe('button');

        await userEvent.click(button);
        expect(onClick).toHaveBeenCalledWith('unblock');
    });

    it('exposes the action through a test id', () => {
        render(() => <ClientBlockMenuItem action="block" onClick={() => {}} />);
        expect(screen.getByTestId('client-block-menu-item')).toBeInTheDocument();
    });

    it('merges the caller class onto the button', () => {
        render(() => (
            <ClientBlockMenuItem action="unblock" onClick={() => {}} class="custom-class" />
        ));
        expect(screen.getByRole('button').className).toContain('custom-class');
    });
});
