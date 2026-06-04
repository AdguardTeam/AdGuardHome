import React from 'react';

import intl from 'panel/common/intl';

import { ServiceIcons, WebService } from './ServiceIcons';

import s from './PersistentClientsTable.module.pcss';

type ServiceCellProps = {
    serviceIds: string[];
    useGlobal: boolean;
    serviceMap: Map<string, WebService>;
};

export const ServiceCell = ({ serviceIds, useGlobal, serviceMap }: ServiceCellProps) => {
    if (useGlobal) {
        return (
            <div className={s.cell}>
                <span className={s.cellLabel}>{intl.getMessage('blocked_services')}</span>
                <div className={s.cellValue}>
                    <span>{intl.getMessage('settings_global')}</span>
                </div>
            </div>
        );
    }

    if (serviceIds.length === 0) {
        return (
            <div className={s.cell}>
                <span className={s.cellLabel}>{intl.getMessage('blocked_services')}</span>
                <div className={s.cellValue}>
                    <span>-</span>
                </div>
            </div>
        );
    }

    return (
        <div className={s.cell}>
            <span className={s.cellLabel}>{intl.getMessage('blocked_services')}</span>
            <div className={s.cellValue}>
                <ServiceIcons serviceIds={serviceIds} serviceMap={serviceMap} />
            </div>
        </div>
    );
};
