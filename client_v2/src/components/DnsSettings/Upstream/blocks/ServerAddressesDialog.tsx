import { untrack, type Accessor, Show } from 'solid-js';
import { dnsConfigState, setDnsConfig } from 'panel/stores/dnsConfig';
import intl from 'panel/common/intl';
import { ConfigDialog } from 'panel/common/ui/ConfigDialog';
import { Textarea } from 'panel/common/controls/Textarea';
import { validateUpstreams } from 'panel/helpers/validators';
import { useField } from 'panel/hooks/useField';
import { Examples } from './Examples';
import { ServerAddressesFileDialog } from './ServerAddressesFileDialog';
import theme from 'panel/lib/theme';

type Props = {
    open: Accessor<boolean>;
    onClose: () => void;
    processing: boolean;
};

export const ServerAddressesDialog = (props: Props) => {
    const hasFile = () => !!dnsConfigState.upstream_dns_file;

    const field = useField<string>(
        () => props.open(),
        () => dnsConfigState.upstream_dns,
        { validate: (v) => (v ? validateUpstreams(v) || '' : '') },
    );

    return (
        <Show
            when={!hasFile()}
            fallback={<ServerAddressesFileDialog open={props.open} onClose={props.onClose} />}
        >
            <ConfigDialog
                open={props.open()}
                title={intl.getMessage('dns_server_addresses_title')}
                description={
                    <>
                        <p>{intl.getMessage('dns_server_addresses_desc')}</p>
                        <p>
                            {intl.getMessage('dns_server_addresses_desc_2', {
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
                                b: (text: string) => (
                                    <a
                                        href="https://link.adtidy.org/forward.html?action=dns_kb_providers&from=ui&app=home"
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
                        setDnsConfig({ upstream_dns: v });
                        untrack(() => props.onClose());
                    });
                }}
                processing={props.processing}
            >
                <div class={theme.form.input}>
                    <Textarea
                        value={field.value()}
                        onChange={(e: Event) =>
                            field.setValue((e.target as HTMLTextAreaElement).value)
                        }
                        onBlur={() => field.validate()}
                        id="upstream_dns"
                        label={intl.getMessage('dns_server_addresses_label')}
                        placeholder={intl.getMessage('dns_server_addresses_placeholder')}
                        errorMessage={field.error()}
                        size="medium"
                    />
                </div>
                <Examples />
            </ConfigDialog>
        </Show>
    );
};
