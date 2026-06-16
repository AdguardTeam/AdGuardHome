import { Show } from 'solid-js';

import intl from 'panel/common/intl';

import { ServiceIcons, type WebService } from './ServiceIcons';

import s from './PersistentClientsTable.module.pcss';

type ServiceCellProps = {
    serviceIds: string[];
    useGlobal: boolean;
    serviceMap: Map<string, WebService>;
};

export const ServiceCell = (props: ServiceCellProps) => {
    return (
        <div class={s.cell}>
            <span class={s.cellLabel}>{intl.getMessage('blocked_services')}</span>
            <div class={s.cellValue}>
                <Show when={props.useGlobal} fallback={
                    <Show when={props.serviceIds.length > 0} fallback={<span>-</span>}>
                        <ServiceIcons serviceIds={props.serviceIds} serviceMap={props.serviceMap} />
                    </Show>
                }>
                    <span>{intl.getMessage('settings_global')}</span>
                </Show>
            </div>
        </div>
    );
};
