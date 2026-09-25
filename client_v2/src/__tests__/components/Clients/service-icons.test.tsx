import { describe, it, expect } from 'vitest';
import { render, screen } from '@solidjs/testing-library';

import {
    ServiceIcons,
    type WebService,
} from 'panel/components/Clients/blocks/PersistentClientsTable/ServiceIcons';

const SVG = '<svg xmlns="http://www.w3.org/2000/svg"><circle r="4" /></svg>';

const makeServiceMap = (ids: string[]) =>
    new Map<string, WebService>(
        ids.map((id) => [
            id,
            { id, name: id, icon_svg: SVG, group_id: 'other', rules: [] as string[] },
        ]),
    );

const renderStrip = (serviceIds: string[], maxVisible?: number) =>
    render(() => (
        <ServiceIcons
            serviceIds={serviceIds}
            serviceMap={makeServiceMap(serviceIds)}
            maxVisible={maxVisible}
        />
    ));

const renderedIcons = () => screen.queryAllByTestId('service-icon');

const badgeCount = () => screen.queryByTestId('services-count')?.textContent ?? null;

const manyServices = (count: number) => Array.from({ length: count }, (_, i) => `service-${i}`);

describe('ServiceIcons', () => {
    it('renders two inline icons and counts the remaining ones', () => {
        renderStrip(manyServices(12));

        expect(renderedIcons()).toHaveLength(2);
        expect(badgeCount()).toBe('10');
    });

    it('keeps the icon count and the badge in sync with the total', () => {
        // The invariant that broke the layout: the strip must never render more
        // icons than it accounts for, otherwise the extra icon overflows the cell.
        renderStrip(manyServices(12));

        const visible = renderedIcons().length;
        expect(visible + Number(badgeCount())).toBe(12);
    });

    it('collapses the third service into the count chip', () => {
        // Regression guard: three inline icons need 102px but the narrowest
        // `blocked_services` cell only provides 96px, so the last icon used to
        // be clipped mid-glyph.
        renderStrip(manyServices(3));

        expect(renderedIcons()).toHaveLength(2);
        expect(badgeCount()).toBe('1');
    });

    it('renders no count chip when every service fits', () => {
        renderStrip(manyServices(2));

        expect(renderedIcons()).toHaveLength(2);
        expect(badgeCount()).toBeNull();
    });

    it('renders a single icon without a count chip', () => {
        renderStrip(manyServices(1));

        expect(renderedIcons()).toHaveLength(1);
        expect(badgeCount()).toBeNull();
    });

    it('renders no icons when there are no blocked services', () => {
        renderStrip([]);

        expect(renderedIcons()).toHaveLength(0);
        expect(badgeCount()).toBeNull();
    });

    it('does not render icons for ids missing from the service map', () => {
        render(() => (
            <ServiceIcons
                serviceIds={['known', 'unknown']}
                serviceMap={makeServiceMap(['known'])}
            />
        ));

        expect(renderedIcons()).toHaveLength(1);
        expect(badgeCount()).toBeNull();
    });

    it('respects an explicit maxVisible override', () => {
        renderStrip(manyServices(12), 1);

        expect(renderedIcons()).toHaveLength(1);
        expect(badgeCount()).toBe('11');
    });

    it('does not truncate the icons when the override exceeds the default', () => {
        renderStrip(manyServices(12), 5);

        expect(renderedIcons()).toHaveLength(5);
        expect(badgeCount()).toBe('7');
    });
});
