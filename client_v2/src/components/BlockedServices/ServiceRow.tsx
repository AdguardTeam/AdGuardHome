import { createMemo } from 'solid-js';
import { Switch } from 'panel/common/controls/Switch';
import { decodeSvg } from 'panel/helpers/helpers';

import s from './BlockedServices.module.pcss';

type Props = {
    id: string;
    name: string;
    iconSvg: string;
    checked: boolean;
    disabled: boolean;
    onChange: (id: string, checked: boolean) => void;
};

export const ServiceRow = (props: Props) => {
    const handleChange = (e: Event) => {
        const target = e.target as HTMLInputElement;
        props.onChange(props.id, target.checked);
    };

    const handleRowClick = () => {
        if (props.disabled) {
            return;
        }
        props.onChange(props.id, !props.checked);
    };

    const decodedSvg = createMemo(() => decodeSvg(props.iconSvg));

    return (
        <div
            class={s.serviceRow}
            onClick={handleRowClick}
            role="button"
            tabIndex={0}
            data-testid={`blocked-service-row-${props.id}`}
            onKeyDown={(e: KeyboardEvent) => {
                if (e.key === 'Enter' || e.key === ' ') {
                    handleRowClick();
                }
            }}
        >
            {/* eslint-disable-next-line solid/no-innerhtml */}
            <div class={s.serviceIcon} innerHTML={decodedSvg()} />
            <div class={s.serviceName}>{props.name}</div>
            <div
                class={s.switchWrap}
                onClick={(e: Event) => e.stopPropagation()}
                role="presentation"
            >
                <Switch
                    id={`service_${props.id}`}
                    checked={props.checked}
                    disabled={props.disabled}
                    onChange={handleChange}
                />
            </div>
        </div>
    );
};
