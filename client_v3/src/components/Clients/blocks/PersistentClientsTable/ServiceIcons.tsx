import { createMemo, Show, For } from 'solid-js';

import { Dropdown } from 'panel/common/ui/Dropdown';
import { decodeSvg } from 'panel/helpers/helpers';

import s from './PersistentClientsTable.module.pcss';

const MAX_VISIBLE_SERVICES = 3;

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
        <div class={s.servicesIcons}>
            <div class={s.servicesIconsList}>
                <For each={visibleIds()}>
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
            <Show when={hiddenCount() > 0}>
                <div class={s.countDropdown}>
                    <Dropdown
                        trigger="hover"
                        noIcon
                        overlayClass={s.servicesTooltipOverlay}
                        menu={
                            <div class={s.servicesTooltip}>
                                <div class={s.servicesTooltipGrid}>
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
                        <span class={s.countLabel}>{hiddenCount()}</span>
                    </Dropdown>
                </div>
            </Show>
        </div>
    );
};
