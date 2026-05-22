import React from 'react';
import cn from 'clsx';

import theme from 'panel/lib/theme';

import { LogEntry, Service } from 'panel/components/QueryLog/types';
import { getResponseDetails, getStatusClassName, getStatusLabel } from 'panel/components/QueryLog/helpers';
import { Filter } from 'panel/helpers/helpers';
import s from '../LogTable.module.pcss';

type Props = {
    row: LogEntry;
    filters: Filter[];
    services: Service[];
    whitelistFilters: Filter[];
};

export const ResponseCell = ({ row, filters, services, whitelistFilters }: Props) => {
    const statusClassName = getStatusClassName(row.reason);
    const statusLabel = getStatusLabel(row.reason, row.originalResponse, false);
    const responseDetails = getResponseDetails({
        elapsedMs: row.elapsedMs,
        filters,
        reason: row.reason,
        rules: row.rules,
        serviceName: row.service_name,
        services,
        whitelistFilters,
    });

    return (
        <div className={s.responseCell}>
            <span className={cn(s.status, statusClassName, theme.text.t3)}>{statusLabel}</span>
            <span className={cn(s.secondaryLine, theme.text.t4)} title={responseDetails}>
                {responseDetails}
            </span>
        </div>
    );
};
