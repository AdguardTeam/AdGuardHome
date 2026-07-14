import { untrack, type Accessor } from 'solid-js';
import { dnsConfigState, setDnsConfig } from 'panel/stores/dnsConfig';
import intl from 'panel/common/intl';
import { ConfigDialog } from 'panel/common/ui/ConfigDialog';
import { Textarea } from 'panel/common/controls/Textarea';
import { validateUpstreams } from 'panel/helpers/validators';
import { useField } from 'panel/hooks/useField';
import { Examples } from './Examples';
import theme from 'panel/lib/theme';

type Props = {
    open: Accessor<boolean>;
    onClose: () => void;
    processing: boolean;
};

export const FallbackDnsDialog = (props: Props) => {
    const field = useField<string>(
        () => props.open(),
        () => dnsConfigState.fallback_dns,
        { validate: (v) => (v ? validateUpstreams(v) || '' : '') },
    );

    return (
        <ConfigDialog
            open={props.open()}
            title={intl.getMessage('dns_fallback_dns_title')}
            description={
                <>
                    <p>{intl.getMessage('dns_fallback_dns_desc')}</p>
                    <p>
                        {intl.getMessage('dns_fallback_dns_desc_2', {
                            a: (text: string) => (
                                <a
                                    href="https://github.com/AdguardTeam/AdGuardHome/wiki/Configuration#upstreams"
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    class={theme.link.link}
                                >
                                    {text}
                                </a>
                            ),
                        })}
                    </p>
                </>
            }
            onClose={props.onClose}
            onSubmit={() => {
                field.submitIfValid((v) => {
                    setDnsConfig({ fallback_dns: v });
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
                    id="fallback_dns"
                    label={intl.getMessage('dns_fallback_dns_label')}
                    placeholder={intl.getMessage('dns_fallback_dns_placeholder')}
                    errorMessage={field.error()}
                    size="medium"
                    highlightComments
                />
            </div>
            <Examples />
        </ConfigDialog>
    );
};
