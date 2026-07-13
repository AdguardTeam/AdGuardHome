import { createSignal, createEffect } from 'solid-js';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { RoutePath } from 'panel/components/Routes/Paths';
import { Link } from 'panel/common/ui/Link';
import { setFiltersConfig } from 'panel/stores/filtering';
import { SettingRow } from 'panel/common/ui/SettingRow';

export type FormValues = {
    enabled: boolean;
    interval: number;
};

type Props = {
    initialValues: FormValues;
    processing: boolean;
};

export const FiltersConfig = (props: Props) => {
    const [enabled, setEnabled] = createSignal(props.initialValues.enabled);

    createEffect(() => {
        const initial = props.initialValues;
        setFiltersConfig({ ...initial, enabled: enabled() });
    });

    return (
        <SettingRow
            variant="switch"
            id="filtering_enabled"
            title={intl.getMessage('settings_filter_requests')}
            description={intl.getMessage('settings_filter_requests_desc', {
                a: (text: string) => (
                    <Link
                        to={RoutePath.DnsBlocklists}
                        class={theme.link.link}
                        onClick={(e: Event) => e.stopPropagation()}
                    >
                        {text}
                    </Link>
                ),
                b: (text: string) => (
                    <Link
                        to={RoutePath.DnsAllowlists}
                        class={theme.link.link}
                        onClick={(e: Event) => e.stopPropagation()}
                    >
                        {text}
                    </Link>
                ),
                c: (text: string) => (
                    <Link
                        to={RoutePath.UserRules}
                        class={theme.link.link}
                        onClick={(e: Event) => e.stopPropagation()}
                    >
                        {text}
                    </Link>
                ),
            })}
            checked={enabled()}
            onChange={(v) => setEnabled(v)}
        />
    );
};
