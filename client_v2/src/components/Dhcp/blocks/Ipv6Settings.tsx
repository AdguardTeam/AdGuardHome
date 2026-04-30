import React, { useState, useEffect } from 'react';
import cn from 'clsx';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { Input } from 'panel/common/controls/Input';
import { Button } from 'panel/common/ui/Button';

import s from '../Dhcp.module.pcss';

type V6Config = {
    range_start: string;
    lease_duration: number;
};

type Props = {
    v6?: V6Config;
    interfaces?: Record<string, any>;
    selectedInterface: string;
    processingConfig: boolean;
    onSave: (values: V6Config) => void;
};

export const Ipv6Settings = ({ v6, interfaces, selectedInterface, processingConfig, onSave }: Props) => {
    const [rangeStart, setRangeStart] = useState(v6?.range_start || '');
    const [leaseDuration, setLeaseDuration] = useState<string>(
        v6?.lease_duration ? String(v6.lease_duration) : '',
    );

    useEffect(() => {
        setRangeStart(v6?.range_start || '');
        setLeaseDuration(v6?.lease_duration ? String(v6.lease_duration) : '');
    }, [v6]);

    const hasIpv6 = !!(interfaces && interfaces[selectedInterface]?.ipv6_addresses);
    const isEmptyConfig = !rangeStart && !leaseDuration;

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        onSave({
            range_start: rangeStart,
            lease_duration: leaseDuration ? Number(leaseDuration) : 0,
        });
    };

    return (
        <form onSubmit={handleSubmit} className={s.form}>
            <div className={cn(theme.form.group, s.formGroup)}>
                <div className={s.formField}>
                    <span className={cn(theme.text.t3, s.formFieldLabel)}>
                        {intl.getMessage('dhcp_form_range_title_v2')}
                    </span>
                    <div className={s.rangeRow}>
                        <Input
                            id="v6_range_start"
                            placeholder={intl.getMessage('dhcp_form_range_start_v2')}
                            value={rangeStart}
                            onChange={(e: React.ChangeEvent<HTMLInputElement>) => setRangeStart(e.target.value)}
                            disabled={!hasIpv6}
                        />
                    </div>
                </div>

                <div className={s.formField}>
                    <Input
                        id="v6_lease_duration"
                        type="number"
                        label={intl.getMessage('dhcp_form_lease_title_v2')}
                        placeholder="86400"
                        value={leaseDuration}
                        onChange={(e: React.ChangeEvent<HTMLInputElement>) => setLeaseDuration(e.target.value)}
                        disabled={!hasIpv6}
                    />
                </div>
            </div>

            <div className={theme.form.buttonGroup}>
                <Button
                    type="submit"
                    variant="primary"
                    size="small"
                    disabled={processingConfig || !hasIpv6 || isEmptyConfig}
                    className={theme.form.button}
                >
                    {intl.getMessage('save')}
                </Button>
            </div>
        </form>
    );
};
