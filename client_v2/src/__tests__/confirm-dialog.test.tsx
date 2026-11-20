import { render, screen } from '@solidjs/testing-library';
import { describe, expect, it } from 'vitest';

import { ConfirmDialog } from 'panel/common/ui/ConfirmDialog';
import theme from 'panel/lib/theme';

describe('ConfirmDialog', () => {
    it('puts a custom footer in the standard footer container', () => {
        render(() => (
            <ConfirmDialog
                onClose={() => {}}
                buttonText="Confirm"
                cancelText="Cancel"
                customFooter={<button type="button">Use defaults</button>}
            />
        ));

        const customButton = screen.getByRole('button', { name: 'Use defaults' });

        expect(customButton.parentElement?.classList.contains(theme.dialog.footer)).toBe(true);
        expect(screen.queryByRole('button', { name: 'Confirm' })).not.toBeInTheDocument();
    });
});
