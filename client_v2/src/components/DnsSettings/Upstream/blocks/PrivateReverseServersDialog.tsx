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

export const PrivateReverseServersDialog = (props: Props) => {
    const field = useField<string>(
        () => props.open(),
        () => dnsConfigState.local_ptr_upstreams,
        { validate: (v) => (v ? validateUpstreams(v) || '' : '') },
    );

    return (
        <ConfigDialog
            open={props.open()}
            title={intl.getMessage('dns_private_reverse_servers_title')}
            description={
                <>
                    <p>{intl.getMessage('dns_private_reverse_servers_desc')}</p>
                    {dnsConfigState.default_local_ptr_upstreams.length >= 2 && (
                        <p>
                            {intl.getMessage('dns_private_reverse_servers_desc_2', {
                                value_1: dnsConfigState.default_local_ptr_upstreams[0],
                                value_2: dnsConfigState.default_local_ptr_upstreams[1],
                            })}
                        </p>
                    )}
                </>
            }
            onClose={props.onClose}
            onSubmit={() => {
                field.submitIfValid((v) => {
                    setDnsConfig({ local_ptr_upstreams: v });
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
                    id="local_ptr_upstreams"
                    label={intl.getMessage('dns_private_reverse_servers_label')}
                    placeholder={intl.getMessage('dns_private_reverse_servers_placeholder')}
                    errorMessage={field.error()}
                    size="medium"
                    highlightComments
                />
            </div>
            <Examples />
        </ConfigDialog>
    );
};
