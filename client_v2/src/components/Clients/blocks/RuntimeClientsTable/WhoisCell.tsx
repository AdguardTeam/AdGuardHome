import React from 'react';
import cn from 'clsx';
import intl from 'panel/common/intl';
import { Icon } from 'panel/common/ui/Icon';
import theme from 'panel/lib/theme';

import { WhoisInfo } from 'panel/initialState';
import s from './RuntimeClientsTable.module.pcss';

type Props = {
    whoisInfo: WhoisInfo;
    ip: string;
};

const stripHtml = (str: string) => str.replace(/<\/?span>/g, '');

export const WhoisCell = ({ whoisInfo, ip }: Props) => {
    const raw = whoisInfo || {};
    const { country } = raw;
    const orgname = raw.orgname || raw.org;
    const hasData = country || orgname;

    if (!hasData) {
        return <span>-</span>;
    }

    return (
        <div className={s.whoisCell}>
            <div className={s.whoisInline}>
                {country && (
                    <span className={s.whoisRow}>
                        <Icon icon="location" color="green" className={s.whoisIcon} />
                        <span className={s.whoisText}>{country}</span>
                    </span>
                )}
                {orgname && (
                    <span className={s.whoisRow}>
                        <Icon icon="wifi" color="green" className={s.whoisIcon} />
                        <span className={cn(theme.common.textOverflow, s.whoisText)}>
                            {orgname}
                        </span>
                    </span>
                )}
            </div>

            <div className={s.tooltip}>
                <div className={s.tooltipTitle}>{intl.getMessage('client_details')}</div>
                <div className={s.tooltipRow}>
                    {stripHtml(
                        intl.getMessage('query_log_detail_address', {
                            value: ip,
                            span: (v: string) => v,
                        }),
                    )}
                </div>
                {country && (
                    <div className={s.tooltipRow}>
                        {stripHtml(
                            intl.getMessage('query_log_detail_country', {
                                value: country,
                                span: (v: string) => v,
                            }),
                        )}
                    </div>
                )}
                {orgname && (
                    <div className={s.tooltipRow}>
                        {stripHtml(
                            intl.getMessage('query_log_detail_network', {
                                value: orgname,
                                span: (v: string) => v,
                            }),
                        )}
                    </div>
                )}
            </div>
        </div>
    );
};
