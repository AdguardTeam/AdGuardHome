import { Show } from 'solid-js';
import { useNavigate } from '@solidjs/router';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { Button } from 'panel/common/ui/Button';
import { Icon } from 'panel/common/ui/Icon';
import theme from 'panel/lib/theme';
import { Paths } from 'panel/components/Routes/Paths';
import { dhcpState, toggleDhcp } from 'panel/stores/dhcp';

import s from './DhcpDisabledBanner.module.pcss';

export const DhcpDisabledBanner = () => {
    const navigate = useNavigate();

    const handleEnable = () => {
        const v4 = dhcpState.v4;
        const hasV4Config = !!(v4 && Object.values(v4).some(Boolean));

        if (hasV4Config) {
            toggleDhcp({
                enabled: false,
                interface_name: dhcpState.interface_name,
                v4,
                v6: dhcpState.v6,
            });
        } else {
            navigate(Paths.Dhcp);
        }
    };

    return (
        <Show when={!dhcpState.enabled && dhcpState.dhcp_available}>
            <div class={s.wrapper}>
                <div class={s.banner} role="status" data-testid="dhcp-disabled-banner">
                    <Icon icon="attention_filled" class={s.icon} />
                    <div class={cn(s.message, theme.text.t2)}>
                        {intl.getMessage('setting_not_applied_dhcp')}
                    </div>
                    <Button
                        variant="secondary"
                        class={s.action}
                        onClick={handleEnable}
                        data-testid="enable-dhcp-button"
                    >
                        {intl.getMessage('enable')}
                    </Button>
                </div>
            </div>
        </Show>
    );
};
