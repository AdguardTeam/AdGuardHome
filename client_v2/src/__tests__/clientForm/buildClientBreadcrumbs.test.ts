import { describe, expect, it } from 'vitest';
import { buildClientBreadcrumbs } from 'panel/helpers/buildClientBreadcrumbs';
import { RoutePath } from 'panel/components/Routes/Paths';

describe('buildClientBreadcrumbs', () => {
    it('builds add-mode breadcrumbs with extra links', () => {
        const result = buildClientBreadcrumbs({ mode: 'add', originalName: '' }, [
            { path: RoutePath.ClientsBlockedServices, title: 'Blocked Services' },
        ]);
        expect(result).toHaveLength(3);
        expect(result[0].path).toBe(RoutePath.Clients);
        expect(result[1].path).toBe(RoutePath.ClientsAdd);
        expect(result[2].path).toBe(RoutePath.ClientsBlockedServices);
    });

    it('builds edit-mode breadcrumbs with encoded clientName', () => {
        const result = buildClientBreadcrumbs({ mode: 'edit', originalName: 'My Client' }, []);
        expect(result).toHaveLength(2);
        expect(result[0].path).toBe(RoutePath.Clients);
        expect(result[1].path).toBe(RoutePath.ClientsEdit);
        expect(result[1].props?.clientName).toBe('My%20Client');
    });

    it('returns empty array when clientForm is null', () => {
        const result = buildClientBreadcrumbs(null, []);
        expect(result).toHaveLength(0);
    });
});
