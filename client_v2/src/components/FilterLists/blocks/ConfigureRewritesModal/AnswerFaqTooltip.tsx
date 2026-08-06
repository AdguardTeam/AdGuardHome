import { For, Show } from 'solid-js';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { FaqTooltip } from 'panel/common/ui/FaqTooltip';
import { Icon } from 'panel/common/ui/Icon';
import theme from 'panel/lib/theme';
import s from './ConfigureRewritesModal.module.pcss';

const items = [
    {
        message: intl.getMessage('rewrites_tooltip_instructions_item1'),
    },
    {
        message: intl.getMessage('rewrites_tooltip_instructions_item2'),
    },
    {
        message: intl.getMessage('rewrites_tooltip_instructions_item3'),
        code: 'A',
    },
    {
        message: intl.getMessage('rewrites_tooltip_instructions_item4'),
        code: 'AAAA',
    },
];

export const AnswerFaqTooltip = () => {
    return (
        <FaqTooltip
            overlayClass={s.dropdown}
            menuClass={s.tooltip}
            text={
                <>
                    <div class={cn(theme.text.t3, s.tooltipTitle)}>
                        {intl.getMessage('instructions')}
                    </div>

                    <For each={items}>
                        {(item) => (
                            <div class={s.tooltipItem}>
                                <Icon icon="label" class={s.icon} />
                                {item.message}

                                <Show when={item.code}>
                                    <code>{item.code}</code>
                                </Show>
                            </div>
                        )}
                    </For>
                </>
            }
        />
    );
};
