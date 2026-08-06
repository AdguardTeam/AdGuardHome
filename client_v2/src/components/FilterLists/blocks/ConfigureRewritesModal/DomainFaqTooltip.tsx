import { For } from 'solid-js';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { FaqTooltip } from 'panel/common/ui/FaqTooltip';
import { Icon } from 'panel/common/ui/Icon';
import theme from 'panel/lib/theme';
import s from './ConfigureRewritesModal.module.pcss';

const items = [
    {
        message: intl.getMessage('rewrites_tooltip_examples_item1'),
        code: 'example.org',
    },
    {
        message: intl.getMessage('rewrites_tooltip_examples_item2'),
        code: '*.example.org',
    },
];

export const DomainFaqTooltip = () => {
    return (
        <FaqTooltip
            overlayClass={s.dropdown}
            menuClass={s.tooltip}
            text={
                <>
                    <div class={cn(theme.text.t3, s.tooltipTitle)}>
                        {intl.getMessage('upstream_examples_title')}
                    </div>

                    <For each={items}>
                        {(item) => (
                            <div class={s.tooltipItem}>
                                <Icon icon="label" class={s.icon} />
                                {item.message}
                                <code>{item.code}</code>
                            </div>
                        )}
                    </For>
                </>
            }
        />
    );
};
