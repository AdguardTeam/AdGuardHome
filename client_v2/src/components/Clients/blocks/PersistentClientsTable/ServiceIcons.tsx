import React from 'react';

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

export const ServiceIcons = ({
    serviceIds,
    serviceMap,
    maxVisible = MAX_VISIBLE_SERVICES,
}: ServiceIconsProps) => {
    const visibleIds = serviceIds.slice(0, maxVisible);
    const hiddenCount = serviceIds.length - maxVisible;

    return (
        <div className={s.servicesIcons}>
            {visibleIds.map((svcId) => {
                const svc = serviceMap.get(svcId);
                if (!svc) return null;
                return (
                    <div
                        key={svcId}
                        className={s.serviceIcon}
                        title={svc.name}
                        dangerouslySetInnerHTML={{
                            __html: decodeSvg(svc.icon_svg),
                        }}
                    />
                );
            })}
            {hiddenCount > 0 && (
                <Dropdown
                    trigger="hover"
                    noIcon
                    overlayClassName={s.servicesTooltipOverlay}
                    menu={
                        <div className={s.servicesTooltip}>
                            <div className={s.servicesTooltipGrid}>
                                {serviceIds.map((svcId) => {
                                    const svc = serviceMap.get(svcId);
                                    if (!svc) return null;
                                    return (
                                        <div
                                            key={svcId}
                                            className={s.serviceIcon}
                                            title={svc.name}
                                            dangerouslySetInnerHTML={{
                                                __html: decodeSvg(svc.icon_svg),
                                            }}
                                        />
                                    );
                                })}
                            </div>
                        </div>
                    }
                >
                    <span className={s.countLabel}>{hiddenCount}</span>
                </Dropdown>
            )}
        </div>
    );
};
