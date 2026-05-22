import React from 'react';
import cn from 'clsx';

import theme from 'panel/lib/theme';
import { Filter } from 'panel/helpers/helpers';
import { LogEntry, Service } from 'panel/components/QueryLog/types';
import {
    getQueryReasonLabel,
    getQueryReasonDetails,
    getQueryReasonKey,
} from 'panel/components/QueryLog/helpers';

import s from '../LogTable.module.pcss';

type Props = {
    row: LogEntry;
    filters: Filter[];
    services: Service[];
    whitelistFilters: Filter[];
};

export const ReasonCell = ({ row, filters, services, whitelistFilters }: Props) => {
    const rules = row.rules ?? [];
    const reasonKey = getQueryReasonKey(row.reason, rules);
    const reasonDetails = getQueryReasonDetails({
        elapsedMs: row.elapsedMs,
        filters,
        reason: row.reason,
        rules,
        serviceName: row.service_name || row.serviceName,
        services,
        whitelistFilters,
    });
    const reasonLabel = getQueryReasonLabel(reasonKey);

    return (
        <div className={s.reasonCell}>
            <span className={cn(theme.text.t3, s.reasonLabel)}>{reasonLabel}</span>
            <span className={cn(s.secondaryLine, theme.text.t4)} title={reasonDetails || undefined}>
                {reasonDetails}
            </span>
        </div>
    );
};
