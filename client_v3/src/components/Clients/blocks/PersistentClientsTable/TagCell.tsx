import { Show } from 'solid-js';

import intl from 'panel/common/intl';

import { TagsRow } from './TagsRow';

import s from './PersistentClientsTable.module.pcss';

type TagCellProps = {
    tags: string[];
    onCopy?: (text: string) => void;
};

export const TagCell = (props: TagCellProps) => {
    return (
        <div class={s.cell}>
            <span class={s.cellLabel}>{intl.getMessage('tags_title')}</span>
            <div class={s.cellValue}>
                <Show when={props.tags.length > 0} fallback={<span>-</span>}>
                    <TagsRow tags={props.tags} onCopy={props.onCopy} />
                </Show>
            </div>
        </div>
    );
};
