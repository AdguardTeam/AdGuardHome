import React from 'react';
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

export const GroupFilter = ({ groups, activeGroups, onToggleGroup }: Props) => {
    if (groups.length === 0) {
        return null;
    }

    return (
        <div className={s.groups}>
            {groups.map((group) => (
                <button
                    key={group.id}
                    type="button"
                    onClick={() => onToggleGroup(group.id)}
                    aria-pressed={activeGroups.includes(group.id)}
                    className={cn(s.groupButton, {
                        [s.groupButtonActive]: activeGroups.includes(group.id),
                    })}
                >
                    {getGroupName(group.id)}
                </button>
            ))}
        </div>
    );
};
