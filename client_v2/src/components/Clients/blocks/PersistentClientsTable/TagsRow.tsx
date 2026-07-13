import { createMemo, Show, For } from 'solid-js';
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

export const TagsRow = (props: TagsRowProps) => {
    const maxVisible = () => props.maxVisible ?? MAX_VISIBLE_TAGS;
    const visible = createMemo(() => props.tags.slice(0, maxVisible()));
    const hiddenCount = createMemo(() => props.tags.length - maxVisible());

    return (
        <div class={s.tagsRow}>
            <span class={s.tagsText}>
                {visible().join(', ')}
                <Show when={hiddenCount() > 0}>,</Show>
            </span>
            <Show when={hiddenCount() > 0}>
                <div class={s.countDropdown}>
                    <Dropdown
                        trigger="hover"
                        noIcon
                        overlayClass={s.tagsTooltipOverlay}
                        menu={
                            <div class={s.tagsTooltip}>
                                <For each={props.tags}>
                                    {(tag) => <span class={s.tagsTooltipItem}>{tag}</span>}
                                </For>
                                <Show when={props.onCopy}>
                                    {(onCopy) => (
                                        <button
                                            type="button"
                                            class={cn(s.copyBtn, s.copyBtnGreen, s.copyBtnTopRight)}
                                            onClick={() => onCopy()(props.tags.join(', '))}
                                            title={intl.getMessage('copy')}
                                        >
                                            <Icon icon="copy" color="green" />
                                        </button>
                                    )}
                                </Show>
                            </div>
                        }
                    >
                        <span class={s.countLabel}>{hiddenCount()}</span>
                    </Dropdown>
                </div>
            </Show>
        </div>
    );
};
