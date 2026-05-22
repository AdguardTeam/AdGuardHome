import React from 'react';
import cn from 'clsx';

import theme from 'panel/lib/theme';
import intl from 'panel/common/intl';

import { Icon } from 'panel/common/ui/Icon';
import s from '../UserRules.module.pcss';

const EXAMPLES = [
    intl.getMessage('user_rules_example_block', { domain: 'example.org' }),
    intl.getMessage('user_rules_example_unblock', { domain: 'example.org' }),
    intl.getMessage('user_rules_example_response', { domain: 'example.org' }),
    intl.getMessage('user_rules_example_comment'),
    intl.getMessage('user_rules_example_comment_2'),
    intl.getMessage('user_rules_example_regex'),
];

export const Examples = () => (
    <div className={s.examplesSection}>
        <h2 className={cn(theme.title.h6, s.sectionTitle)}>{intl.getMessage('upstream_examples_title')}</h2>
        <ul className={s.examplesList}>
            {EXAMPLES.map((example, index) => (
                <li key={index} className={cn(theme.text.t3, s.listItem)}>
                    <Icon icon="label" className={s.icon} />

                    {example}
                </li>
            ))}
        </ul>

        <p className={cn(s.learnMore, theme.text.t2)}>
            {intl.getMessage('user_rules_learn_more', {
                a: (text: string) => (
                    <a
                        href="https://link.adtidy.org/forward.html?action=dns_kb_filtering_syntax&from=ui&app=home"
                        target="_blank"
                        rel="noopener noreferrer"
                    >
                        {text}
                    </a>
                ),
            })}
        </p>
    </div>
);
