import { untrack, type Accessor } from 'solid-js';
import { dnsConfigState, setDnsConfig } from 'panel/stores/dnsConfig';
import intl from 'panel/common/intl';
import { ConfigDialog } from 'panel/common/ui/ConfigDialog';
import { Input } from 'panel/common/controls/Input';
import { UPSTREAM_TIMEOUT } from 'panel/helpers/constants';
import { validateBetween } from 'panel/helpers/validators';
import { useField } from 'panel/hooks/useField';
import theme from 'panel/lib/theme';

type Props = {
    open: Accessor<boolean>;
    onClose: () => void;
    processing: boolean;
};

export const TimeoutDialog = (props: Props) => {
    const field = useField<number>(
        () => props.open(),
        () => dnsConfigState.upstream_timeout,
        {
            validate: (v) =>
                (Number.isNaN(v)
                    ? intl.getMessage('form_error_required')
                    : validateBetween(v, UPSTREAM_TIMEOUT.MIN, UPSTREAM_TIMEOUT.MAX)) || '',
        },
    );

    return (
        <ConfigDialog
            open={props.open()}
            title={intl.getMessage('dns_upstream_timeout')}
            description={intl.getMessage('dns_upstream_timeout_desc')}
            onClose={props.onClose}
            onSubmit={() => {
                field.submitIfValid((v) => {
                    setDnsConfig({ upstream_timeout: v });
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
                        field.setValue(
                            (e.target as HTMLInputElement).value === ''
                                ? NaN
                                : Number((e.target as HTMLInputElement).value),
                        )
                    }
                    onBlur={() => field.validate()}
                    id="upstream_timeout"
                    label={intl.getMessage('upstream_timeout')}
                    placeholder={intl.getMessage('dns_upstream_timeout_placeholder')}
                    min={UPSTREAM_TIMEOUT.MIN}
                    max={UPSTREAM_TIMEOUT.MAX}
                    errorMessage={field.error()}
                    size="large"
                />
            </div>
        </ConfigDialog>
    );
};
