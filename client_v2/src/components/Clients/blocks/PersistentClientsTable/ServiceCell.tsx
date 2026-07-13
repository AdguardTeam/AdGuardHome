import { Show } from 'solid-js';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';

import { ServiceIcons, type WebService } from './ServiceIcons';

type ServiceCellProps = {
    serviceIds: string[];
    useGlobal: boolean;
    serviceMap: Map<string, WebService>;
};

export const ServiceCell = (props: ServiceCellProps) => {
    return (
        <div class={theme.table.cell}>
            <span class={theme.table.cellLabel}>{intl.getMessage('blocked_services')}</span>
            <div class={theme.table.cellValueText}>
                <Show
                    when={props.useGlobal}
                    fallback={
                        <Show when={props.serviceIds.length > 0} fallback={<span>-</span>}>
                            <ServiceIcons
                                serviceIds={props.serviceIds}
                                serviceMap={props.serviceMap}
                            />
                        </Show>
                    }
                >
                    <span>{intl.getMessage('settings_global')}</span>
                </Show>
            </div>
        </div>
    );
};
