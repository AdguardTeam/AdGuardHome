import { render, screen } from '@solidjs/testing-library';
import { describe, expect, it, vi } from 'vitest';

import { GroupFilter } from 'panel/components/BlockedServices/GroupFilter';

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

        expect(screen.getByTestId('blocked-services-group-gaming')).toHaveTextContent(
            'Gaming',
        );
        expect(screen.getByTestId('blocked-services-group-streaming')).toHaveTextContent(
            'Streaming services',
        );
    });

    it('falls back to the group ID for an unknown group', () => {
        renderGroupFilter(['unknown_group']);

        expect(screen.getByTestId('blocked-services-group-unknown_group')).toHaveTextContent(
            'unknown_group',
        );
    });
});
