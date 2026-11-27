import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@solidjs/testing-library';

import { ServiceCell } from 'panel/components/Clients/blocks/PersistentClientsTable/ServiceCell';
import type { WebService } from 'panel/components/Clients/blocks/PersistentClientsTable/ServiceIcons';
import { copyInDom } from 'panel/__tests__/helpers/copy';

vi.mock('panel/common/intl', async () =>
    (await import('panel/__tests__/helpers/copy')).createIntlMock(),
);

const SVG = '<svg xmlns="http://www.w3.org/2000/svg"><circle r="4" /></svg>';

const serviceMap = new Map<string, WebService>([
    ['youtube', { id: 'youtube', name: 'YouTube', icon_svg: SVG, group_id: 'streaming', rules: [] }],
]);

describe('ServiceCell — blocked services status (AGH-234)', () => {
    it('shows Global when the client uses the global blocked services', () => {
        render(() => <ServiceCell serviceIds={['youtube']} useGlobal serviceMap={serviceMap} />);

        expect(screen.getByText(copyInDom('settings_global'))).toBeInTheDocument();
        expect(screen.queryByTestId('service-icon')).toBeNull();
    });

    it('shows the service icons for custom blocked services', () => {
        render(() => (
            <ServiceCell serviceIds={['youtube']} useGlobal={false} serviceMap={serviceMap} />
        ));

        expect(screen.getByTestId('service-icon')).toBeInTheDocument();
        expect(screen.queryByText(copyInDom('settings_global'))).toBeNull();
    });

    it('shows a dash when the client has no custom services', () => {
        render(() => <ServiceCell serviceIds={[]} useGlobal={false} serviceMap={serviceMap} />);

        expect(screen.getByText('-')).toBeInTheDocument();
        expect(screen.queryByTestId('service-icon')).toBeNull();
    });
});
