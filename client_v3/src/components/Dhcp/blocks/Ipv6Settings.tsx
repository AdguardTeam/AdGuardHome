import { createSignal, createEffect, createMemo } from 'solid-js';
import cn from 'clsx';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { Input } from 'panel/common/controls/Input';
import { Button } from 'panel/common/ui/Button';
import type { DhcpInterfaces } from 'panel/initialState';
import { validateIpv6 } from 'panel/helpers/validators';

import s from '../Dhcp.module.pcss';

type V6Config = {
    range_start: string;
    lease_duration: number;
};

type Props = {
    v6?: V6Config;
    interfaces?: DhcpInterfaces;
    selectedInterface: string;
    processingConfig: boolean;
    onSave: (values: V6Config) => void;
};

export const Ipv6Settings = (props: Props) => {
    const [rangeStart, setRangeStart] = createSignal('');
    const [leaseDuration, setLeaseDuration] = createSignal('');

    const [rangeStartError, setRangeStartError] = createSignal('');

    // Sync with props
    createEffect(() => {
        setRangeStart(props.v6?.range_start || '');
        setLeaseDuration(props.v6?.lease_duration ? String(props.v6.lease_duration) : '');
    });

    const hasIpv6 = createMemo(
        () => !!(props.interfaces && props.interfaces[props.selectedInterface]?.ipv6_addresses),
    );

    const isEmptyConfig = createMemo(() => !rangeStart() && !leaseDuration());

    const validateRangeStart = () => {
        const err = validateIpv6(rangeStart());
        setRangeStartError(err || '');
    };

    const handleSubmit = (e: Event) => {
        e.preventDefault();
        validateRangeStart();

        if (rangeStartError()) {
            return;
        }

        props.onSave({
            range_start: rangeStart().trim(),
            lease_duration: leaseDuration() ? Number(leaseDuration()) : 0,
        });
    };

    return (
        <form onSubmit={handleSubmit} class={s.form}>
            <div class={cn(theme.form.group, s.formGroup)}>
                <div class={s.formField}>
                    <span class={cn(theme.text.t3, s.formFieldLabel)}>
                        {intl.getMessage('dhcp_form_range_title')}
                    </span>
                    <div class={s.formField}>
                        <Input
                            id="v6_range_start"
                            placeholder={intl.getMessage('dhcp_form_range_start')}
                            value={rangeStart()}
                            onChange={(e: Event) =>
                                setRangeStart((e.target as HTMLInputElement).value)
                            }
                            onBlur={validateRangeStart}
                            errorMessage={rangeStartError()}
                            disabled={!hasIpv6()}
                        />
                    </div>
                </div>

                <div class={s.formField}>
                    <Input
                        id="v6_lease_duration"
                        type="number"
                        label={intl.getMessage('dhcp_form_lease_title')}
                        placeholder="86400"
                        value={leaseDuration()}
                        onChange={(e: Event) =>
                            setLeaseDuration((e.target as HTMLInputElement).value)
                        }
                        disabled={!hasIpv6()}
                    />
                </div>
            </div>

            <div class={theme.form.buttonGroup}>
                <Button
                    type="submit"
                    variant="primary"
                    size="small"
                    disabled={props.processingConfig || !hasIpv6() || isEmptyConfig()}
                    class={theme.form.button}
                >
                    {intl.getMessage('save')}
                </Button>
            </div>
        </form>
    );
};
