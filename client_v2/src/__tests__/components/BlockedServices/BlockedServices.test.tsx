import { render, screen, fireEvent } from '@solidjs/testing-library';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, createMemoryHistory } from '@solidjs/router';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { BlockedServices } from 'panel/components/BlockedServices/BlockedServices';
import { clearClientForm, clientFormState, initClientForm } from 'panel/stores/clientForm';
import { copyInDom } from 'panel/__tests__/helpers/copy';

const ALL_SERVICES = [
    {
        id: 'telegram',
        name: 'Telegram',
        icon_svg: '<svg><circle cx="12" cy="12" r="10"/></svg>',
        group_id: 'messaging',
        rules: ['||telegram.org^'],
    },
    {
        id: 'whatsapp',
        name: 'WhatsApp',
        icon_svg: '<svg><circle cx="12" cy="12" r="10"/></svg>',
        group_id: 'messaging',
        rules: ['||whatsapp.com^'],
    },
];

const servicesMocks = vi.hoisted(() => ({
    getBlockedServices: vi.fn(),
    getAllBlockedServices: vi.fn(),
    updateBlockedServices: vi.fn(),
    servicesState: {
        allServices: [] as unknown[],
        allGroups: [] as unknown[],
        list: { ids: [] as string[] } as { ids: string[]; schedule?: unknown },
        processingAll: false,
        processing: false,
        processingSet: false,
        processingServiceId: null as string | null,
        initialized: false,
        allInitialized: false,
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

describe('BlockedServices global update pending state', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        clearClientForm();
        servicesMocks.servicesState.allServices = ALL_SERVICES;
        servicesMocks.servicesState.list = { ids: [] };
        servicesMocks.servicesState.processing = false;
        servicesMocks.servicesState.processingSet = false;
        servicesMocks.servicesState.processingServiceId = null;
    });

    it('disables every global switch and marks the list busy during the PUT', () => {
        servicesMocks.servicesState.processingSet = true;
        renderBlockedServices(false);

        expect(document.getElementById('service_telegram')).toBeDisabled();
        expect(document.getElementById('service_whatsapp')).toBeDisabled();
        expect(screen.getByTestId('blocked-services-list')).toHaveAttribute(
            'aria-busy',
            'true',
        );
        expect(screen.getByTestId('blocked-service-row-telegram')).toHaveAttribute(
            'aria-disabled',
            'true',
        );
        expect(document.querySelector('label[for="service_telegram"]')?.className).toMatch(
            /_busy_/,
        );
    });

    it('shows the disabled palette only on the switch being saved', () => {
        servicesMocks.servicesState.processingSet = true;
        servicesMocks.servicesState.processingServiceId = 'telegram';
        renderBlockedServices(false);

        expect(document.querySelector('label[for="service_telegram"]')?.className).not.toMatch(
            /_busy_/,
        );
        expect(document.querySelector('label[for="service_whatsapp"]')?.className).toMatch(
            /_busy_/,
        );
    });

    it('keeps the switches disabled while the follow-up read is in flight', () => {
        servicesMocks.servicesState.processing = true;
        renderBlockedServices(false);

        expect(document.getElementById('service_telegram')).toBeDisabled();
        expect(screen.getByTestId('blocked-services-list')).toHaveAttribute(
            'aria-busy',
            'true',
        );
    });

    it('rejects a toggle that arrives after an update has started', async () => {
        renderBlockedServices(false);

        await fireEvent.click(screen.getByTestId('blocked-service-row-telegram'));
        expect(servicesMocks.updateBlockedServices).toHaveBeenCalledTimes(1);

        servicesMocks.servicesState.processingSet = true;
        await fireEvent.click(screen.getByTestId('blocked-service-row-whatsapp'));

        expect(servicesMocks.updateBlockedServices).toHaveBeenCalledTimes(1);
    });

    it('keeps per-client switches editable while a global update is pending', async () => {
        initClientForm({
            name: 'Laptop',
            use_global_blocked_services: false,
            blocked_services: [],
        });
        servicesMocks.servicesState.processingSet = true;
        renderBlockedServices(true);

        const telegram = document.getElementById('service_telegram') as HTMLInputElement;
        expect(telegram).not.toBeDisabled();

        await userEvent.click(telegram);

        expect(clientFormState.blocked_services).toContain('telegram');
        expect(servicesMocks.updateBlockedServices).not.toHaveBeenCalled();
    });
});
