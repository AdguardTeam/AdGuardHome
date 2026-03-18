import React, { MouseEvent } from 'react';
import cn from 'clsx';

import theme from 'panel/lib/theme';
import { LogEntry } from 'panel/components/QueryLog/types';

import { WhoisInfo } from '../../WhoisInfo';
import s from '../LogTable.module.pcss';

type Props = {
    onSearchSelect: (value: string) => (event: MouseEvent<HTMLButtonElement>) => void;
    row: LogEntry;
};

export const ClientCell = ({ onSearchSelect, row }: Props) => {
    const clientName = row.client_info?.name || '';

    return (
        <div className={s.clientCell}>
            <div className={s.clientPrimary}>
                <button
                    type="button"
                    className={cn(s.clientLink, s.clientIp, theme.text.t3)}
                    title={row.client}
                    onClick={onSearchSelect(row.client)}
                >
                    {row.client}
                </button>
            </div>
            <div className={s.clientSecondary}>
                {clientName && (
                    <button
                        type="button"
                        className={cn(s.clientLink, s.clientName, theme.text.t4)}
                        title={clientName}
                        onClick={onSearchSelect(clientName)}
                    >
                        {clientName}
                    </button>
                )}
                <WhoisInfo whois={row.client_info?.whois} />
            </div>
        </div>
    );
};
