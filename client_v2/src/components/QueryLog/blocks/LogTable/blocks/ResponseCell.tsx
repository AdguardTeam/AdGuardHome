import cn from 'clsx';

import theme from 'panel/lib/theme';

import { LogEntry, Service } from 'panel/components/QueryLog/types';
import {
    getResponseDetails,
    getStatusClassName,
    getStatusLabel,
} from 'panel/components/QueryLog/helpers';
import { Filter } from 'panel/helpers/helpers';
import s from '../LogTable.module.pcss';

type Props = {
    row: LogEntry;
    filters: Filter[];
    services: Service[];
    whitelistFilters: Filter[];
};

export const ResponseCell = (props: Props) => {
    const statusClassName = () => getStatusClassName(props.row.reason);
    const statusLabel = () => getStatusLabel(props.row.reason, props.row.originalResponse, false);
    const responseDetails = () =>
        getResponseDetails({
            elapsedMs: props.row.elapsedMs,
            filters: props.filters,
            reason: props.row.reason,
            rules: props.row.rules,
            serviceName: props.row.service_name,
            services: props.services,
            whitelistFilters: props.whitelistFilters,
        });

    return (
        <div class={s.responseCell}>
            <span class={cn(s.status, statusClassName(), theme.text.t3)}>{statusLabel()}</span>
            <span class={cn(s.secondaryLine, theme.text.t4)} title={responseDetails()}>
                {responseDetails()}
            </span>
        </div>
    );
};
