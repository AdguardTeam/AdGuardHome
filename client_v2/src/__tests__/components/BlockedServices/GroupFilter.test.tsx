import { render, screen } from '@solidjs/testing-library';
import { describe, expect, it, vi } from 'vitest';

import { GroupFilter } from 'panel/components/BlockedServices/GroupFilter';
import { copyInDom } from 'panel/__tests__/helpers/copy';

vi.mock('panel/common/intl', async () => {
    const { en } = await import('panel/__tests__/helpers/copy');

    return {
        default: {
            getMessage: (key: string) =>
                (en as Record<string, string>)[key] ?? key,
        },
    };
});

const renderGroupFilter = (ids: string[]) =>
    render(() => (
        <GroupFilter
            groups={ids.map((id) => ({ id }))}
            activeGroups={[]}
            onToggleGroup={() => {}}
        />
    ));

describe('GroupFilter', () => {
    it('uses the Hostlists Registry group names', () => {
        renderGroupFilter(['gaming', 'streaming']);

        const gaming = screen.getByTestId('blocked-services-group-gaming');
        expect(gaming).toHaveTextContent(copyInDom('servicesgroup.gaming.name'));

        const streaming = screen.getByTestId('blocked-services-group-streaming');
        expect(streaming).toHaveTextContent(copyInDom('servicesgroup.streaming.name'));
    });

    it('falls back to the group ID for an unknown group', () => {
        renderGroupFilter(['unknown_group']);

        const unknown = screen.getByTestId('blocked-services-group-unknown_group');
        expect(unknown).toHaveTextContent('unknown_group');
    });
});
