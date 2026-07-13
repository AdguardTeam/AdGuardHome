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

import { SettingRow } from 'panel/common/ui/SettingRow';

describe('SettingRow', () => {
    describe('switch variant', () => {
        it('renders title and description', () => {
            render(() => (
                <SettingRow
                    id="test-switch"
                    variant="switch"
                    title="Test Title"
                    description="Test Description"
                />
            ));
            expect(screen.getByText('Test Title')).toBeDefined();
            expect(screen.getByText('Test Description')).toBeDefined();
        });

        it('fires onChange when row is clicked', () => {
            const onChange = vi.fn();
            render(() => (
                <SettingRow
                    id="test-switch"
                    variant="switch"
                    title="Title"
                    checked={false}
                    onChange={onChange}
                />
            ));
            const row = screen.getByRole('button');
            fireEvent.click(row);
            expect(onChange).toHaveBeenCalledTimes(1);
            expect(onChange).toHaveBeenCalledWith(true);
        });

        it('fires onChange when switch is toggled off', () => {
            const onChange = vi.fn();
            render(() => (
                <SettingRow
                    id="test-switch"
                    variant="switch"
                    title="Title"
                    checked={true}
                    onChange={onChange}
                />
            ));
            const row = screen.getByRole('button');
            fireEvent.click(row);
            expect(onChange).toHaveBeenCalledWith(false);
        });

        it('does not fire onChange when disabled', () => {
            const onChange = vi.fn();
            render(() => (
                <SettingRow
                    id="test-switch"
                    variant="switch"
                    title="Title"
                    checked={false}
                    disabled={true}
                    onChange={onChange}
                />
            ));
            const row = screen.getByRole('button');
            fireEvent.click(row);
            expect(onChange).not.toHaveBeenCalled();
        });
    });

    describe('link variant', () => {
        it('renders value in semibold', () => {
            render(() => (
                <SettingRow
                    id="test-link"
                    variant="link"
                    title="Retention"
                    value="90 days · 3 ignored domains"
                    onClick={vi.fn()}
                />
            ));
            expect(screen.getByText('90 days · 3 ignored domains')).toBeDefined();
        });

        it('fires onClick when link is clicked', () => {
            const onClick = vi.fn();
            render(() => (
                <SettingRow
                    id="test-link"
                    variant="link"
                    title="Retention"
                    value="Summary"
                    onClick={onClick}
                />
            ));
            const rows = screen.getAllByRole('button');
            fireEvent.click(rows[0]);
            expect(onClick).toHaveBeenCalledTimes(1);
        });

        it('does not fire onClick when disabled', () => {
            const onClick = vi.fn();
            render(() => (
                <SettingRow
                    id="test-link"
                    variant="link"
                    title="Retention"
                    value="Summary"
                    disabled={true}
                    onClick={onClick}
                />
            ));
            const rows = screen.getAllByRole('button');
            fireEvent.click(rows[0]);
            expect(onClick).not.toHaveBeenCalled();
        });
    });

    describe('switch-link variant', () => {
        it('renders switch without arrow or configure label', () => {
            render(() => (
                <SettingRow
                    id="test-combo"
                    variant="switch-link"
                    title="Safe Search"
                    description="Block inappropriate content"
                    checked={true}
                    value="Enabled · 4 providers"
                    onChange={vi.fn()}
                    onClick={vi.fn()}
                />
            ));
            expect(screen.queryByText('settings_configure')).toBeNull();
            expect(screen.getByText('Enabled · 4 providers')).toBeDefined();
        });

        it('row click fires onClick, not onChange', () => {
            const onChange = vi.fn();
            const onClick = vi.fn();
            render(() => (
                <SettingRow
                    id="test-combo"
                    variant="switch-link"
                    title="Title"
                    checked={true}
                    onChange={onChange}
                    onClick={onClick}
                />
            ));
            const rows = screen.getAllByRole('button');
            fireEvent.click(rows[0]);
            expect(onClick).toHaveBeenCalledTimes(1);
            expect(onChange).not.toHaveBeenCalled();
        });

        it('disabled suppresses switch', () => {
            const onChange = vi.fn();
            const onClick = vi.fn();
            render(() => (
                <SettingRow
                    id="test-combo"
                    variant="switch-link"
                    title="Title"
                    checked={false}
                    disabled={true}
                    onChange={onChange}
                    onClick={onClick}
                />
            ));
            const rows = screen.getAllByRole('button');
            fireEvent.click(rows[0]);
            expect(onChange).not.toHaveBeenCalled();
            expect(onClick).not.toHaveBeenCalled();
        });
    });

    describe('children slot', () => {
        it('renders children when provided', () => {
            render(() => (
                <SettingRow id="test-children" variant="switch" title="Title">
                    <div data-testid="child-content">Child Content</div>
                </SettingRow>
            ));
            expect(screen.getByTestId('child-content')).toBeDefined();
            expect(screen.getByText('Child Content')).toBeDefined();
        });
    });
});
