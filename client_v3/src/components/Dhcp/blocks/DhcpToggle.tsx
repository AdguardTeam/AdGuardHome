import type { Accessor } from 'solid-js';
import { SettingRow } from 'panel/common/ui/SettingRow';
import intl from 'panel/common/intl';
import { dhcpState, toggleDhcp } from 'panel/stores/dhcp';

type Props = {
    selectedInterface: Accessor<string>;
    onToggleOn?: () => void;
};

type DhcpToggleConfig = {
    enabled: boolean;
    interface_name: string;
    v4?: typeof dhcpState.v4;
    v6?: typeof dhcpState.v6;
};

export const DhcpToggle = (props: Props) => {
    const disabled = () =>
        dhcpState.processingDhcp ||
        dhcpState.processingConfig ||
        (!dhcpState.enabled && !props.selectedInterface());

    const onChange = (checked: boolean) => {
        if (checked) {
            const v4 = dhcpState.v4;
            const v6 = dhcpState.v6;
            const hasV4Config = !!(v4 && Object.values(v4).some(Boolean));
            const hasV6Config = !!(v6 && Object.values(v6).some(Boolean));

            const values: DhcpToggleConfig = {
                enabled: false,
                interface_name: props.selectedInterface() || dhcpState.interface_name,
            };
            if (hasV4Config) values.v4 = v4;
            if (hasV6Config) values.v6 = v6;

            toggleDhcp(values);
            if (!hasV4Config) {
                props.onToggleOn?.();
            }
        } else {
            toggleDhcp({ enabled: true });
        }
    };

    return (
        <SettingRow
            id="dhcp_enabled"
            variant="switch"
            title={intl.getMessage('dhcp_enable')}
            checked={!!dhcpState.enabled}
            disabled={disabled()}
            onChange={onChange}
        />
    );
};
