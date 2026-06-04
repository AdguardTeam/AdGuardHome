import React from 'react';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { Dropdown } from 'panel/common/ui/Dropdown';
import { Icon } from 'panel/common/ui/Icon';

import s from './PersistentClientsTable.module.pcss';

const MAX_VISIBLE_TAGS = 1;

type TagsRowProps = {
    tags: string[];
    maxVisible?: number;
    onCopy?: (text: string) => void;
};

export const TagsRow = ({ tags, maxVisible = MAX_VISIBLE_TAGS, onCopy }: TagsRowProps) => {
    const visible = tags.slice(0, maxVisible);
    const hiddenCount = tags.length - maxVisible;

    return (
        <div className={s.tagsRow}>
            <span className={s.tagsText}>
                {visible.join(', ')}
                {hiddenCount > 0 && ','}
            </span>
            {hiddenCount > 0 && (
                <Dropdown
                    trigger="hover"
                    noIcon
                    overlayClassName={s.tagsTooltipOverlay}
                    menu={
                        <div className={s.tagsTooltip}>
                            {tags.map((tag) => (
                                <span key={tag} className={s.tagsTooltipItem}>
                                    {tag}
                                </span>
                            ))}
                            {onCopy && (
                                <button
                                    type="button"
                                    className={cn(s.copyBtn, s.copyBtnGreen, s.copyBtnTopRight)}
                                    onClick={() => onCopy(tags.join(', '))}
                                    title={intl.getMessage('copy')}
                                >
                                    <Icon icon="copy" color="green" />
                                </button>
                            )}
                        </div>
                    }
                >
                    <span className={s.countLabel}>{hiddenCount}</span>
                </Dropdown>
            )}
        </div>
    );
};
