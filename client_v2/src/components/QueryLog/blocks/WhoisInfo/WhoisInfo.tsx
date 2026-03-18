import React from 'react';
import cn from 'clsx';

import theme from 'panel/lib/theme';
import { normalizeWhois } from 'panel/helpers/helpers';
import { Icon } from 'panel/common/ui/Icon';
import { WhoisInfo as WhoisInfoType } from 'panel/components/QueryLog/types';

import s from './WhoisInfo.module.pcss';

type Props = {
    whois: WhoisInfoType | undefined | null;
    className?: string;
};

export const WhoisInfo = ({ whois, className }: Props) => {
    if (!whois || Object.keys(whois).length === 0) {
        return null;
    }

    const whoisInfo = normalizeWhois(whois);
    const entries = Object.entries(whoisInfo).filter(([, value]) => Boolean(value));

    return (
        <div className={cn(s.whoisInfo, theme.text.t4, className)}>
            {entries.map(([key, value], index) => (
                <React.Fragment key={key}>
                    <span className={s.whoisItem} title={String(value)}>
                        {key === 'location' && <Icon icon="location" className={s.whoisIcon} />}
                        {(key === 'orgname' || key === 'netname' || key === 'descr') && (
                            <Icon icon="wifi" className={s.whoisIcon} />
                        )}
                        <span className={s.whoisText}>{value}</span>
                    </span>
                    {index < entries.length - 1 && (
                        <span className={s.divider} />
                    )}
                </React.Fragment>
            ))}
        </div>
    );
};
