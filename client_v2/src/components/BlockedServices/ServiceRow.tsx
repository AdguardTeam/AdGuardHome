import React, { ChangeEvent } from 'react';
import { Switch } from 'panel/common/controls/Switch';

import s from './BlockedServices.module.pcss';

type Props = {
    id: string;
    name: string;
    iconSvg: string;
    checked: boolean;
    disabled: boolean;
    onChange: (id: string, checked: boolean) => void;
};

export const ServiceRow = ({ id, name, iconSvg, checked, disabled, onChange }: Props) => {
    const handleChange = (e: ChangeEvent<HTMLInputElement>) => {
        onChange(id, e.target.checked);
    };

    const handleRowClick = () => {
        if (disabled) {
            return;
        }
        onChange(id, !checked);
    };

    const decodedSvg = (() => {
        try {
            return atob(iconSvg);
        } catch {
            return '';
        }
    })();

    return (
        <div
            className={s.serviceRow}
            onClick={handleRowClick}
            role="button"
            tabIndex={0}
            onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') handleRowClick(); }}
        >
            <div
                className={s.serviceIcon}
                dangerouslySetInnerHTML={{ __html: decodedSvg }}
            />
            <div className={s.serviceName}>{name}</div>
            <div className={s.switchWrap} onClick={(e) => e.stopPropagation()} role="presentation">
                <Switch
                    id={`service_${id}`}
                    checked={checked}
                    disabled={disabled}
                    onChange={handleChange}
                />
            </div>
        </div>
    );
};
