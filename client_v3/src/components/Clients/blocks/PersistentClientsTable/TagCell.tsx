import { Show } from 'solid-js';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';

import { TagsRow } from './TagsRow';

type TagCellProps = {
    tags: string[];
    onCopy?: (text: string) => void;
};

export const TagCell = (props: TagCellProps) => {
    return (
        <div class={theme.table.cell}>
            <span class={theme.table.cellLabel}>{intl.getMessage('tags_title')}</span>
            <div class={theme.table.cellValueText}>
                <Show when={props.tags.length > 0} fallback={<span>-</span>}>
                    <TagsRow tags={props.tags} onCopy={props.onCopy} />
                </Show>
            </div>
        </div>
    );
};
