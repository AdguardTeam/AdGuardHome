import { render, screen } from '@solidjs/testing-library';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { InactivitySchedule } from 'panel/components/BlockedServices/InactivitySchedule/InactivitySchedule';
import { clearClientForm, initClientForm } from 'panel/stores/clientForm';
import { copyInDom } from 'panel/__tests__/helpers/copy';

vi.mock('panel/common/intl', async () =>
    (await import('panel/__tests__/helpers/copy')).createIntlMock(),
);

const renderSchedule = () => render(() => <InactivitySchedule clientScope />);

describe('InactivitySchedule client scope', () => {
    beforeEach(() => {
        clearClientForm();
    });

    it('is read-only while the client uses the global blocked services', () => {
        initClientForm({
            name: 'Laptop',
            use_global_blocked_services: true,
            blocked_services_schedule: { time_zone: 'UTC' },
        });
        renderSchedule();

        expect(screen.getByTestId('inactivity-schedule-read-only-note')).toHaveTextContent(
            copyInDom('clients_global_inactivity_schedule_note'),
        );
        expect(screen.queryByTestId('schedule-row-mon-add')).toBeNull();
        expect(screen.queryByTestId('schedule-row-mon-edit')).toBeNull();
        expect(screen.queryByTestId('schedule-row-mon-delete')).toBeNull();
    });

    it('is editable while the client uses its own blocked services', () => {
        initClientForm({
            name: 'Laptop',
            use_global_blocked_services: false,
            blocked_services_schedule: { time_zone: 'UTC' },
        });
        renderSchedule();

        expect(screen.queryByTestId('inactivity-schedule-read-only-note')).toBeNull();
        expect(screen.getByTestId('schedule-row-mon-add')).toBeInTheDocument();
    });
});
