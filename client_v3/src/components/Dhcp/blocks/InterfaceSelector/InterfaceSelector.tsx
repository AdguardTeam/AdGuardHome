import type { Accessor } from 'solid-js';
import cn from 'clsx';
import { Show } from 'solid-js';
import { useNavigate } from '@solidjs/router';
import { Select } from 'panel/common/controls/Select';
import { Button } from 'panel/common/ui/Button';
import intl from 'panel/common/intl';
import { dhcpState, findActiveDhcp } from 'panel/stores/dhcp';
import s from './InterfaceSelector.module.pcss';
import theme from 'panel/lib/theme';

type Props = {
    selectedInterface: Accessor<string>;
    onInterfaceChange: (name: string) => void;
    showAllIps: Accessor<boolean>;
    onShowAllIps: () => void;
};

const MAX_VISIBLE_IPS = 2;

export const InterfaceSelector = (props: Props) => {
    const navigate = useNavigate();
    const interfaces = () => dhcpState.interfaces || {};

    const selectOptions = () =>
        Object.keys(interfaces()).map((name) => {
            const iface = interfaces()[name];
            const ipv4 = iface?.ipv4_addresses?.join(', ') || '';
            const ipv6 = iface?.ipv6_addresses?.join(', ') || '';
            let label = name;
            if (ipv4) label += ` - ${ipv4}`;
            if (ipv6) label += ` - ${ipv6}`;
            return { label, value: name };
        });

    const selectedIface = () => interfaces()[props.selectedInterface()];

    const gatewayIp = () => selectedIface()?.gateway_ip || '';
    const hardwareAddress = () => selectedIface()?.hardware_address || '';
    const ipAddresses = () => selectedIface()?.ip_addresses || [];

    const displayIps = () => {
        const ips = ipAddresses();
        if (ips.length <= MAX_VISIBLE_IPS || props.showAllIps()) return ips;
        return ips.slice(0, MAX_VISIBLE_IPS);
    };

    const remainingCount = () => {
        const ips = ipAddresses();
        if (ips.length <= MAX_VISIBLE_IPS || props.showAllIps()) return 0;
        return ips.length - MAX_VISIBLE_IPS;
    };

    const handleCheck = () => {
        if (props.selectedInterface()) {
            findActiveDhcp(props.selectedInterface(), navigate);
        }
    };

    const selectedOption = () => selectOptions().find((o) => o.value === props.selectedInterface());

    return (
        <div class={s.section}>
            <div class={s.selectWrap}>
                <div class={cn(s.label, theme.text.t3)}>
                    {intl.getMessage('dhcp_interface_select')}
                </div>
                <Select
                    options={selectOptions()}
                    value={selectedOption()}
                    onChange={(option: { value: string }) => props.onInterfaceChange(option.value)}
                    size="responsive"
                    height="big"
                />
            </div>
            <Show when={props.selectedInterface()}>
                <div class={s.info}>
                    <Show when={gatewayIp()}>
                        <div class={s.row}>
                            {intl.getMessage('dhcp_form_gateway_address_value', {
                                value: gatewayIp(),
                            })}
                        </div>
                    </Show>
                    <Show when={hardwareAddress()}>
                        <div class={s.row}>
                            {intl.getMessage('dhcp_hardware_address_value', {
                                value: hardwareAddress(),
                            })}
                        </div>
                    </Show>
                    <Show when={ipAddresses().length > 0}>
                        <div class={s.row}>
                            {intl.getMessage('dhcp_ip_addresses_value', {
                                value: displayIps().join(', '),
                            })}

                            <Show when={remainingCount() > 0}>
                                <button
                                    type="button"
                                    class={s.interfaceInfoMore}
                                    onClick={() => props.onShowAllIps()}
                                >
                                    {intl.getMessage('show_more_count', {
                                        count: String(remainingCount()),
                                    })}
                                </button>
                            </Show>
                        </div>
                    </Show>
                </div>
                <div class={s.buttonWrap}>
                    <Button
                        variant="primary"
                        size="small"
                        onClick={handleCheck}
                        disabled={dhcpState.processingDhcp}
                        class={s.button}
                        compact
                    >
                        {intl.getMessage('check_dhcp_servers')}
                    </Button>
                </div>
            </Show>
        </div>
    );
};
