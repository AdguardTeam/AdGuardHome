import { untrack, type Accessor } from 'solid-js';

import { dnsConfigState, setDnsConfig } from 'panel/stores/dnsConfig';
import intl from 'panel/common/intl';
import { ConfigDialog } from 'panel/common/ui/ConfigDialog';
import { Input } from 'panel/common/controls/Input';
import { IPV4_SUBNET_PREFIX, IP_VERSION } from 'panel/helpers/constants';
import { validateRequiredValue, validateBetween } from 'panel/helpers/validators';
import { useField } from 'panel/hooks/useField';
import theme from 'panel/lib/theme';

type Props = {
    open: Accessor<boolean>;
    onClose: () => void;
    processing: boolean;
};

export const SubnetPrefixV4Dialog = (props: Props) => {
    const field = useField<number>(
        () => props.open(),
        () => dnsConfigState.ratelimit_subnet_len_ipv4 ?? IPV4_SUBNET_PREFIX.DEFAULT,
        {
            validate: (v) =>
                validateRequiredValue(String(v)) ||
                validateBetween(v, IPV4_SUBNET_PREFIX.MIN, IPV4_SUBNET_PREFIX.MAX) ||
                '',
        },
    );

    return (
        <ConfigDialog
            open={props.open()}
            title={intl.getMessage('dns_subnet_prefix_title', { value: IP_VERSION.V4 })}
            description={intl.getMessage('dns_subnet_prefix_desc', { value: IP_VERSION.V4 })}
            onClose={props.onClose}
            onSubmit={() => {
                field.submitIfValid((v) => {
                    setDnsConfig({ ratelimit_subnet_len_ipv4: v });
                    untrack(() => props.onClose());
                });
            }}
            processing={props.processing}
        >
            <div class={theme.form.input}>
                <Input
                    type="number"
                    value={field.value()}
                    onChange={(e: Event) =>
                        field.setValue(Number((e.target as HTMLInputElement).value))
                    }
                    onBlur={() => field.validate()}
                    id="ratelimit_subnet_len_ipv4"
                    label={intl.getMessage('dns_subnet_prefix_title', { value: IP_VERSION.V4 })}
                    placeholder={intl.getMessage('dns_subnet_placeholder')}
                    min={IPV4_SUBNET_PREFIX.MIN}
                    max={IPV4_SUBNET_PREFIX.MAX}
                    errorMessage={field.error()}
                    size="large"
                />
            </div>
        </ConfigDialog>
    );
};
