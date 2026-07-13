import { Show, For } from 'solid-js';
import cn from 'clsx';
import intl from 'panel/common/intl';

import s from './BlockedServices.module.pcss';

interface ServiceGroup {
    id: string;
}

type Props = {
    groups: ServiceGroup[];
    activeGroups: string[];
    onToggleGroup: (groupId: string) => void;
};

const getGroupName = (id: string): string => {
    switch (id) {
        case 'ai':
            return intl.getMessage('parental_group_ai');
        case 'cdn':
            return intl.getMessage('parental_group_cdn');
        case 'dating':
            return intl.getMessage('parental_group_dating');
        case 'gambling':
            return intl.getMessage('parental_group_gambling');
        case 'gaming':
            return intl.getMessage('parental_group_gaming');
        case 'messaging':
            return intl.getMessage('parental_group_messaging');
        case 'privacy':
            return intl.getMessage('parental_group_privacy');
        case 'shopping':
            return intl.getMessage('parental_group_shopping');
        case 'social':
            return intl.getMessage('parental_group_social');
        case 'development':
            return intl.getMessage('parental_group_development');
        case 'streaming':
            return intl.getMessage('parental_group_streaming');
        case 'hosting':
            return intl.getMessage('parental_group_hosting');
        case 'messenger':
            return intl.getMessage('parental_group_messenger');
        case 'social_network':
            return intl.getMessage('parental_group_social_network');
        case 'software':
            return intl.getMessage('parental_group_software');
        default:
            return id;
    }
};

export const GroupFilter = (props: Props) => {
    return (
        <Show when={props.groups.length > 0}>
            <div class={s.groups}>
                <For each={props.groups}>
                    {(group) => (
                        <button
                            type="button"
                            onClick={() => props.onToggleGroup(group.id)}
                            aria-pressed={props.activeGroups.includes(group.id)}
                            data-testid={`blocked-services-group-${group.id}`}
                            class={cn(s.groupButton, {
                                [s.groupButtonActive]: props.activeGroups.includes(group.id),
                            })}
                        >
                            {getGroupName(group.id)}
                        </button>
                    )}
                </For>
            </div>
        </Show>
    );
};
