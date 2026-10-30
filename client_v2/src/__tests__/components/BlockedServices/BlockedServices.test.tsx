import { render, screen } from '@solidjs/testing-library';
import { MemoryRouter, Route, createMemoryHistory } from '@solidjs/router';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { BlockedServices } from 'panel/components/BlockedServices/BlockedServices';
import { clearClientForm, initClientForm } from 'panel/stores/clientForm';
import { copyInDom } from 'panel/__tests__/helpers/copy';

const servicesMocks = vi.hoisted(() => ({
    getBlockedServices: vi.fn(),
    getAllBlockedServices: vi.fn(),
    updateBlockedServices: vi.fn(),
    servicesState: {
        allServices: [] as unknown[],
        allGroups: [] as unknown[],
        processingAll: false,
        processing: false,
        processingSet: false,
    },
}));

vi.mock('panel/stores/services', () => servicesMocks);

vi.mock('panel/common/intl', async () =>
    (await import('panel/__tests__/helpers/copy')).createIntlMock(),
);

const renderBlockedServices = (clientScope = true) => {
    const history = createMemoryHistory();
    history.set({ value: '/' });

    return render(() => (
        <MemoryRouter history={history}>
            <Route path="/" component={() => <BlockedServices clientScope={clientScope} />} />
        </MemoryRouter>
    ));
};

describe('BlockedServices client scope', () => {
    beforeEach(() => {
        clearClientForm();
    });

    it('hides the list and shows the global note when global blocked services are on', () => {
        initClientForm({ name: 'Laptop', use_global_blocked_services: true });
        renderBlockedServices();

        expect(screen.queryByTestId('blocked-services-list')).toBeNull();
        expect(screen.queryByTestId('blocked-services-search')).toBeNull();
        expect(screen.getByTestId('blocked-services-global-note')).toHaveTextContent(
            copyInDom('clients_global_blocked_services_note'),
        );

        const scheduleLink = screen.getByTestId('blocked-services-schedule-link');
        expect(scheduleLink.tagName).toBe('DIV');
        expect(scheduleLink).toHaveTextContent(
            copyInDom('clients_global_inactivity_schedule_note'),
        );
        // The arrow stays visible in the disabled row; only its color changes.
        expect(scheduleLink.querySelector('svg')).not.toBeNull();

        expect(document.getElementById('use-global-blocked-services')).toBeChecked();
    });

    it('keeps the list editable when global blocked services are off', () => {
        initClientForm({ name: 'Laptop', use_global_blocked_services: false });
        renderBlockedServices();

        expect(screen.getByTestId('blocked-services-list')).toBeInTheDocument();
        expect(screen.getByTestId('blocked-services-search')).toBeInTheDocument();
        expect(screen.queryByTestId('blocked-services-global-note')).toBeNull();
        expect(screen.getByTestId('blocked-services-schedule-link').tagName).toBe('A');
        expect(document.getElementById('use-global-blocked-services')).not.toBeChecked();
    });

    it('does not render the toggle or the global note outside the client scope', () => {
        renderBlockedServices(false);

        expect(document.getElementById('use-global-blocked-services')).toBeNull();
        expect(screen.queryByTestId('blocked-services-global-note')).toBeNull();
        expect(screen.getByTestId('blocked-services-list')).toBeInTheDocument();
    });
});
