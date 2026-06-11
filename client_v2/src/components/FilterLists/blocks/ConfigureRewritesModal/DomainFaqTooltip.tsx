import React from 'react';
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
            overlayClassName={s.dropdown}
            menuClassName={s.tooltip}
            text={
                <>
                    <div className={cn(theme.text.t3, s.tooltipTitle)}>
                        {intl.getMessage('upstream_examples_title')}
                    </div>

                    {items.map((item, index) => (
                        <div key={index} className={s.tooltipItem}>
                            <Icon icon="label" className={s.icon} />
                            {item.message}
                            <code>{item.code}</code>
                        </div>
                    ))}
                </>
            }
        />
    );
};
