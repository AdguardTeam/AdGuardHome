import { createMemo, Show, For } from 'solid-js';

import { Tooltip } from 'panel/common/ui/Tooltip';
import { decodeSvg } from 'panel/helpers/helpers';

import s from './PersistentClientsTable.module.pcss';

/**
 * Number of icons rendered inline before the rest collapse into the "+N" chip.
 * Two is the largest value that fits the narrowest `blocked_services` cell.
 */
const MAX_VISIBLE_SERVICES = 2;

export type WebService = {
    id: string;
    name: string;
    icon_svg: string;
    group_id: string;
    rules: string[];
};

type ServiceIconsProps = {
    serviceIds: string[];
    serviceMap: Map<string, WebService>;
    maxVisible?: number;
};

export const ServiceIcons = (props: ServiceIconsProps) => {
    const maxVisible = () => props.maxVisible ?? MAX_VISIBLE_SERVICES;
    const visibleIds = createMemo(() => props.serviceIds.slice(0, maxVisible()));
    const hiddenCount = createMemo(() => props.serviceIds.length - maxVisible());

    return (
        <div class={s.servicesIcons} data-testid="service-icons">
            <div class={s.servicesIconsList}>
                <For each={visibleIds()}>
                    {(svcId) => {
                        const svc = props.serviceMap.get(svcId);
                        if (!svc) return null;
                        /* eslint-disable solid/no-innerhtml */
                        return (
                            <div
                                class={s.serviceIcon}
                                data-testid="service-icon"
                                title={svc.name}
                                innerHTML={decodeSvg(svc.icon_svg)}
                            />
                        );
                        /* eslint-enable solid/no-innerhtml */
                    }}
                </For>
            </div>
            <Show when={hiddenCount() > 0}>
                <div class={s.countDropdown}>
                    <Tooltip
                        overlayClass={s.servicesTooltipOverlay}
                        content={
                            <div class={s.servicesTooltip}>
                                <div
                                    class={s.servicesTooltipGrid}
                                    data-testid="services-tooltip-grid"
                                >
                                    <For each={props.serviceIds}>
                                        {(svcId) => {
                                            const svc = props.serviceMap.get(svcId);
                                            if (!svc) return null;
                                            /* eslint-disable solid/no-innerhtml */
                                            return (
                                                <div
                                                    class={s.serviceIcon}
                                                    title={svc.name}
                                                    innerHTML={decodeSvg(svc.icon_svg)}
                                                />
                                            );
                                            /* eslint-enable solid/no-innerhtml */
                                        }}
                                    </For>
                                </div>
                            </div>
                        }
                    >
                        <span class={s.countLabel} data-testid="services-count">
                            {hiddenCount()}
                        </span>
                    </Tooltip>
                </div>
            </Show>
        </div>
    );
};
