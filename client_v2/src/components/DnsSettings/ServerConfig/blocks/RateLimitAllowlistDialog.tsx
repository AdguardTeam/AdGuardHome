import { untrack, type Accessor } from 'solid-js';

import { dnsConfigState, setDnsConfig } from 'panel/stores/dnsConfig';
import intl from 'panel/common/intl';
import { ConfigDialog } from 'panel/common/ui/ConfigDialog';
import { Textarea } from 'panel/common/controls/Textarea';
import { validateIpPerLine } from 'panel/helpers/validators';
import { useField } from 'panel/hooks/useField';
import theme from 'panel/lib/theme';

type Props = {
    open: Accessor<boolean>;
    onClose: () => void;
    processing: boolean;
};

export const RateLimitAllowlistDialog = (props: Props) => {
    const field = useField<string>(
        () => props.open(),
        () => dnsConfigState.ratelimit_whitelist,
        {
            validate: (v) => (v ? validateIpPerLine(v) || '' : ''),
        },
    );

    return (
        <ConfigDialog
            open={props.open()}
            title={intl.getMessage('dns_rate_limit_allowlist_title')}
            description={intl.getMessage('dns_rate_limit_allowlist_desc')}
            onClose={props.onClose}
            onSubmit={() => {
                field.submitIfValid((v) => {
                    setDnsConfig({ ratelimit_whitelist: v });
                    untrack(() => props.onClose());
                });
            }}
            processing={props.processing}
        >
            <div class={theme.form.input}>
                <Textarea
                    value={field.value()}
                    onChange={(e: Event) => field.setValue((e.target as HTMLTextAreaElement).value)}
                    onBlur={() => field.validate()}
                    id="ratelimit_whitelist"
                    label={intl.getMessage('dns_rate_limit_allowlist_label')}
                    placeholder={intl.getMessage('dns_rate_limit_allowlist_placeholder')}
                    size="medium"
                    errorMessage={field.error()}
                />
            </div>
        </ConfigDialog>
    );
};
