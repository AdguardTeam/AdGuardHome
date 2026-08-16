import type { Accessor } from 'solid-js';
import { createSignal, createEffect } from 'solid-js';
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
    /**
     * Shadows dhcpState.enabled with {@code equals: false} so we can
     * force a DOM re-sync even when the value is unchanged (e.g. reverting
     * after an error or when opening the config modal without saving).
     * Synced from store only when {@code processingConfig} is false
     * to avoid flashing during async save.
     */
    const [dhcpEnabled, setDhcpEnabled] = createSignal(false, {
        equals: false,
    });

    createEffect(() => {
        if (!dhcpState.processingConfig) {
            setDhcpEnabled(!!dhcpState.enabled);
        }
    });

    const disabled = () =>
        dhcpState.processingDhcp ||
        dhcpState.processingConfig ||
        (!dhcpState.enabled && !props.selectedInterface());

    const onChange = (checked: boolean) => {
        if (!checked) {
            // Turning OFF — save immediately (no prerequisites needed).
            setDhcpEnabled(false);
            toggleDhcp({ enabled: true });
            return;
        }

        // Turning ON — check prerequisites before saving to backend.
        const v4 = dhcpState.v4;
        const v6 = dhcpState.v6;
        const hasV4Config = !!(v4 && Object.values(v4).some(Boolean));
        const hasV6Config = !!(v6 && Object.values(v6).some(Boolean));

        // v4 config is already set up — save the change.
        if (hasV4Config) {
            const values: DhcpToggleConfig = {
                enabled: false,
                interface_name: props.selectedInterface() || dhcpState.interface_name,
            };
            if (hasV4Config) values.v4 = v4;
            if (hasV6Config) values.v6 = v6;
            toggleDhcp(values);
            return;
        }

        // v4 config is missing — revert UI and open the config modal
        // WITHOUT saving to backend (avoids showing an error toast).
        setDhcpEnabled(false);
        props.onToggleOn?.();
    };

    return (
        <SettingRow
            id="dhcp_enabled"
            variant="switch"
            title={intl.getMessage('dhcp_enable')}
            checked={dhcpEnabled()}
            disabled={disabled()}
            onChange={onChange}
        />
    );
};
