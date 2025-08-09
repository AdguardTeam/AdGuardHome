import React, { useEffect, useRef } from 'react';
import { Controller, useForm } from 'react-hook-form';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { SwitchGroup } from '../SettingsGroup';
import { RoutePath } from 'panel/components/Routes/Paths';
import { Link } from 'panel/common/ui/Link';

export type FormValues = {
    enabled: boolean;
    interval: number;
};

type Props = {
    initialValues: FormValues;
    setFiltersConfig: (values: FormValues) => void;
    processing: boolean;
};

export const FiltersConfig = ({ initialValues, setFiltersConfig, processing }: Props) => {
    const { watch, control, setValue } = useForm({
        mode: 'onBlur',
        defaultValues: initialValues,
    });

    const enabled = watch('enabled');

    useEffect(() => {
        setFiltersConfig({ ...initialValues, enabled });
    }, [enabled]);

    return (
        <div>
            <Controller
                name="enabled"
                control={control}
                render={() => (
                    <SwitchGroup
                        title={intl.getMessage('settings_filter_requests')}
                        description={intl.getMessage('settings_filter_requests_desc', {
                            a: (text: string) => (
                                <Link
                                    to={RoutePath.DnsBlocklists}
                                    className={theme.link.link}
                                    onClick={(e) => e.stopPropagation()}>
                                    {text}
                                </Link>
                            ),
                            b: (text: string) => (
                                <Link
                                    to={RoutePath.DnsAllowlists}
                                    className={theme.link.link}
                                    onClick={(e) => e.stopPropagation()}>
                                    {text}
                                </Link>
                            ),
                            c: (text: string) => (
                                <Link
                                    to={RoutePath.UserRules}
                                    className={theme.link.link}
                                    onClick={(e) => e.stopPropagation()}>
                                    {text}
                                </Link>
                            ),
                        })}
                        disabled={processing}
                        onChange={(e) => setValue('enabled', e.target.checked)}
                        id="filters_enabled"
                        checked={enabled}
                    />
                )}
            />
        </div>
    );
};
