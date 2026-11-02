import { render, screen } from '@solidjs/testing-library';
import { describe, expect, it, vi } from 'vitest';

import { ScheduleRow } from 'panel/components/BlockedServices/InactivitySchedule/ScheduleRow';

vi.mock('panel/common/intl', async () =>
    (await import('panel/__tests__/helpers/copy')).createIntlMock(),
);

type RowProps = {
    data?: { start: number; end: number };
    disabled?: boolean;
};

const renderRow = (props: RowProps = {}) =>
    render(() => (
        <ScheduleRow
            day="mon"
            data={props.data}
            disabled={props.disabled}
            onEdit={vi.fn()}
            onDelete={vi.fn()}
            onAdd={vi.fn()}
        />
    ));

describe('ScheduleRow', () => {
    it('hides the add action when disabled', () => {
        renderRow({ disabled: true });

        expect(screen.queryByTestId('schedule-row-mon-add')).toBeNull();
    });

    it('hides the edit and delete actions when disabled', () => {
        renderRow({ data: { start: 0, end: 3600000 }, disabled: true });

        expect(screen.queryByTestId('schedule-row-mon-edit')).toBeNull();
        expect(screen.queryByTestId('schedule-row-mon-delete')).toBeNull();
        expect(screen.getByTestId('schedule-row-mon-time')).toHaveTextContent('00:00');
    });

    it('renders the add action when enabled', () => {
        renderRow();

        expect(screen.getByTestId('schedule-row-mon-add')).toBeInTheDocument();
    });

    it('renders the edit and delete actions when enabled', () => {
        renderRow({ data: { start: 0, end: 3600000 } });

        expect(screen.getByTestId('schedule-row-mon-edit')).toBeInTheDocument();
        expect(screen.getByTestId('schedule-row-mon-delete')).toBeInTheDocument();
    });
});
