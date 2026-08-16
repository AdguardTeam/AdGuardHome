import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent, screen } from '@solidjs/testing-library';

const { themeMock } = vi.hoisted(() => {
    const proxy: any = new Proxy(
        {},
        {
            get: (_target, prop) => {
                if (prop === Symbol.toPrimitive || prop === 'toString') {
                    return () => '';
                }
                return proxy;
            },
        },
    );
    return { themeMock: proxy };
});

vi.mock('panel/lib/theme', () => ({
    default: themeMock,
}));

vi.mock('panel/common/intl', () => ({
    default: {
        getMessage: (key: string, _values?: Record<string, unknown>) => key,
    },
}));

import { ConfigDialog } from 'panel/common/ui/ConfigDialog';

describe('ConfigDialog', () => {
    it('renders children when open', () => {
        render(() => (
            <ConfigDialog open={true} title="Test Dialog" onClose={vi.fn()} onSubmit={vi.fn()}>
                <div data-testid="body-content">Body Content</div>
            </ConfigDialog>
        ));
        expect(screen.getByText('Body Content')).toBeDefined();
        expect(screen.getByText('Test Dialog')).toBeDefined();
    });

    it('save disabled when processing is true', () => {
        render(() => (
            <ConfigDialog
                open={true}
                title="Test"
                onClose={vi.fn()}
                onSubmit={vi.fn()}
                processing={true}
            >
                <div>Content</div>
            </ConfigDialog>
        ));
        const saveButton = screen.getByTestId('config-dialog-save');
        expect(saveButton).toBeDisabled();
    });

    it('save disabled when submitDisabled is true', () => {
        render(() => (
            <ConfigDialog
                open={true}
                title="Test"
                onClose={vi.fn()}
                onSubmit={vi.fn()}
                submitDisabled={true}
            >
                <div>Content</div>
            </ConfigDialog>
        ));
        const saveButton = screen.getByTestId('config-dialog-save');
        expect(saveButton).toBeDisabled();
    });

    it('clicking save fires onSubmit', () => {
        const onSubmit = vi.fn();
        render(() => (
            <ConfigDialog open={true} title="Test" onClose={vi.fn()} onSubmit={onSubmit}>
                <div>Content</div>
            </ConfigDialog>
        ));
        const saveButton = screen.getByTestId('config-dialog-save');
        fireEvent.click(saveButton);
        expect(onSubmit).toHaveBeenCalledTimes(1);
    });

    it('renders footer before save button', () => {
        render(() => (
            <ConfigDialog
                open={true}
                title="Test"
                onClose={vi.fn()}
                onSubmit={vi.fn()}
                footer={<button data-testid="secondary-action">Secondary</button>}
            >
                <div>Content</div>
            </ConfigDialog>
        ));
        expect(screen.getByTestId('secondary-action')).toBeDefined();
        // The footer div contains both secondary and save button
        const footer = screen.getByTestId('secondary-action').parentElement;
        expect(footer?.querySelector('[data-testid="config-dialog-save"]')).toBeDefined();
    });

    it('processing disables fieldset', () => {
        render(() => (
            <ConfigDialog
                open={true}
                title="Test"
                onClose={vi.fn()}
                onSubmit={vi.fn()}
                processing={true}
            >
                <input data-testid="test-input" />
            </ConfigDialog>
        ));
        const fieldset = screen.getByTestId('test-input').closest('fieldset');
        expect(fieldset).toBeDisabled();
    });
});
