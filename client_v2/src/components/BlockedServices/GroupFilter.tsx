import { Show, For, type JSX } from 'solid-js';
import cn from 'clsx';
import { SERVICE_GROUP_MESSAGES } from 'panel/common/intl/services.groups.generated';
import theme from 'panel/lib/theme';

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
 * Returns the localized name of a blocked-service group, falling back to the
 * raw ID for a group this build has no name for.
 */
const getGroupName = (id: string): string => SERVICE_GROUP_MESSAGES[id]?.() ?? id;

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
                            class={cn(s.groupButton, theme.text.t3, {
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
