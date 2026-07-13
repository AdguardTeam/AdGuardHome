import { createSignal, createEffect, createMemo } from 'solid-js';
import type { Accessor } from 'solid-js';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { ConfigDialog } from 'panel/common/ui/ConfigDialog';
import { Input } from 'panel/common/controls/Input';
import { dhcpState } from 'panel/stores/dhcp';
import { validateIpv6, validateLeaseTime } from 'panel/helpers/validators';

export type V6Config = {
    range_start: string;
    lease_duration: number;
};

type Props = {
    open: boolean;
    selectedInterface: Accessor<string>;
    onClose: () => void;
    onSave: (values: V6Config) => void;
};

export const DhcpV6Modal = (props: Props) => {
    const [rangeStart, setRangeStart] = createSignal('');
    const [leaseDuration, setLeaseDuration] = createSignal('');
    const [rangeStartError, setRangeStartError] = createSignal('');
    const [leaseDurationError, setLeaseDurationError] = createSignal('');

    createEffect(() => {
        if (props.open) {
            const v6 = dhcpState.v6;
            setRangeStart(v6?.range_start || '');
            setLeaseDuration(v6?.lease_duration ? String(v6.lease_duration) : '');
            setRangeStartError('');
            setLeaseDurationError('');
        }
    });

    const hasIpv6 = createMemo(
        () =>
            !!(
                dhcpState.interfaces &&
                dhcpState.interfaces[props.selectedInterface()]?.ipv6_addresses
            ),
    );

    const isEmptyConfig = createMemo(() => !rangeStart() && !leaseDuration());

    const validateRangeStart = () => {
        const err = validateIpv6(rangeStart());
        setRangeStartError(err || '');
    };

    const validateLeaseDuration = () => {
        const err = validateLeaseTime(leaseDuration());
        setLeaseDurationError(err || '');
    };

    const handleSave = () => {
        validateRangeStart();
        validateLeaseDuration();
        if (rangeStartError() || leaseDurationError()) {
            return;
        }
        props.onSave({
            range_start: rangeStart().trim(),
            lease_duration: leaseDuration() ? Number(leaseDuration()) : 0,
        });
    };

    return (
        <ConfigDialog
            open={props.open}
            title={intl.getMessage('dhcp_ipv6_settings')}
            onClose={props.onClose}
            onSubmit={handleSave}
            processing={!!dhcpState.processingConfig}
            submitDisabled={!hasIpv6() || isEmptyConfig()}
        >
            <div class={theme.form.input}>
                <Input
                    id="v6_range_start"
                    label={intl.getMessage('dhcp_form_range_title')}
                    placeholder={intl.getMessage('dhcp_form_range_start')}
                    value={rangeStart()}
                    onChange={(e: Event) => setRangeStart((e.target as HTMLInputElement).value)}
                    onBlur={validateRangeStart}
                    errorMessage={rangeStartError()}
                />
            </div>

            <div class={theme.form.input}>
                <Input
                    id="v6_lease_duration"
                    type="number"
                    label={intl.getMessage('dhcp_form_lease_title')}
                    placeholder="86400"
                    value={leaseDuration()}
                    onChange={(e: Event) => setLeaseDuration((e.target as HTMLInputElement).value)}
                    onBlur={() => validateLeaseDuration()}
                    inputError={leaseDurationError()}
                />
            </div>
        </ConfigDialog>
    );
};
