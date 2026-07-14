import { untrack, type Accessor } from 'solid-js';

import { dnsConfigState, setDnsConfig } from 'panel/stores/dnsConfig';
import intl from 'panel/common/intl';
import { ConfigDialog } from 'panel/common/ui/ConfigDialog';
import { Input } from 'panel/common/controls/Input';
import { RATE_LIMIT } from 'panel/helpers/constants';
import { validateRequiredValue, validateBetween } from 'panel/helpers/validators';
import { useField } from 'panel/hooks/useField';
import theme from 'panel/lib/theme';

type Props = {
    open: Accessor<boolean>;
    onClose: () => void;
    processing: boolean;
};

export const RateLimitDialog = (props: Props) => {
    const field = useField<number>(
        () => props.open(),
        () => dnsConfigState.ratelimit,
        {
            validate: (v) =>
                validateRequiredValue(String(v)) ||
                validateBetween(v, RATE_LIMIT.MIN, RATE_LIMIT.MAX) ||
                '',
        },
    );

    return (
        <ConfigDialog
            open={props.open()}
            title={intl.getMessage('dns_rate_limit_title')}
            description={
                <>
                    <p>{intl.getMessage('dns_rate_limit_desc')}</p>
                    <p>{intl.getMessage('dns_rate_limit_desc_2')}</p>
                </>
            }
            onClose={props.onClose}
            onSubmit={() => {
                field.submitIfValid((v) => {
                    setDnsConfig({ ratelimit: v });
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
                    id="ratelimit"
                    label={intl.getMessage('server_config_rate_limit')}
                    placeholder={intl.getMessage('dns_rate_limit_placeholder')}
                    min={RATE_LIMIT.MIN}
                    max={RATE_LIMIT.MAX}
                    errorMessage={field.error()}
                    size="large"
                />
            </div>
        </ConfigDialog>
    );
};
