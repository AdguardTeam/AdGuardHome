import React from 'react';
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
            overlayClassName={s.dropdown}
            menuClassName={s.tooltip}
            text={
                <>
                    <div className={cn(theme.text.t3, s.tooltipTitle)}>
                        {intl.getMessage('instructions')}
                    </div>

                    {items.map((item, index) => (
                        <div key={index} className={s.tooltipItem}>
                            <Icon icon="label" className={s.icon} />
                            {item.message}

                            {item.code && <code>{item.code}</code>}
                        </div>
                    ))}
                </>
            }
        />
    );
};
