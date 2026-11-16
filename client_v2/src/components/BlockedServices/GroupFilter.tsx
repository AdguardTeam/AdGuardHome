import { Show, For, type JSX } from 'solid-js';
import cn from 'clsx';
import intl from 'panel/common/intl';
import { SERVICE_GROUP_KEYS } from 'panel/common/intl/locales.generated';

import s from './BlockedServices.module.pcss';

interface ServiceGroup {
    id: string;
}

type Props = {
    groups: ServiceGroup[];
    activeGroups: string[];
    onToggleGroup: (groupId: string) => void;
    children?: JSX.Element;
};

/**
 * Returns the localized name of a blocked-service group.  The names are owned
 * by Hostlists Registry and translated in a separate Twosky project.
 */
const getGroupName = (id: string): string => {
    const key = SERVICE_GROUP_KEYS[id];

    return key ? intl.getMessage(key) : id;
};

export const GroupFilter = (props: Props) => {
    return (
        <div class={s.groups}>
            {props.children}
            <Show when={props.groups.length > 0}>
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
            </Show>
        </div>
    );
};
