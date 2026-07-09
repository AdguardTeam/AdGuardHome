import { type Accessor } from 'solid-js';
import { dnsConfigState } from 'panel/stores/dnsConfig';
import intl from 'panel/common/intl';
import { ConfigDialog } from 'panel/common/ui/ConfigDialog';
import theme from 'panel/lib/theme';

type Props = {
    open: Accessor<boolean>;
    onClose: () => void;
};

export const ServerAddressesFileDialog = (props: Props) => (
    <ConfigDialog
        open={props.open()}
        title={intl.getMessage('dns_server_addresses_title')}
        description={
            <p class={theme.common.breakWord}>
                {intl.getMessage('dns_server_addresses_configured_in_file', {
                    path: dnsConfigState.upstream_dns_file,
                })}
            </p>
        }
        onClose={props.onClose}
        onSubmit={props.onClose}
        buttonText={intl.getMessage('close')}
    />
);
