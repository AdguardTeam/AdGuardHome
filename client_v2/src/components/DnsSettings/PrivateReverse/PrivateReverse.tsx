import { createMemo } from 'solid-js';
import cn from 'clsx';

import {
    dnsConfigState,
    togglePrivatePtrResolvers,
    toggleResolveClients,
} from 'panel/stores/dnsConfig';
import intl from 'panel/common/intl';
import { Breadcrumbs } from 'panel/common/ui/Breadcrumbs';
import { SettingRow } from 'panel/common/ui/SettingRow';
import { RoutePath } from 'panel/components/Routes/Paths';
import { useDialog } from 'panel/hooks/useDialog';
import { PrivateReverseServersDialog } from '../Upstream/blocks/PrivateReverseServersDialog';
import { getUpstreamServersSummary } from '../helpers';
import theme from 'panel/lib/theme';

import s from './PrivateReverse.module.pcss';

export const PrivateReverse = () => {
    const serversDialog = useDialog();

    const processing = () => dnsConfigState.processingSetConfig;

    const privateReverseValue = createMemo(() =>
        getUpstreamServersSummary(dnsConfigState.local_ptr_upstreams),
    );

    return (
        <div class={cn(theme.layout.container, s.container)}>
            <div class={cn(theme.layout.containerIn, theme.layout.containerIn_one_col)}>
                <div class={s.breadcrumbs}>
                    <Breadcrumbs
                        parentLinks={[
                            {
                                path: RoutePath.Dns,
                                title: intl.getMessage('dns_settings'),
                            },
                        ]}
                        currentTitle={intl.getMessage('dns_private_reverse_resolvers')}
                    />
                </div>

                <div class={s.form}>
                    <SettingRow
                        variant="switch"
                        id="use_private_ptr_resolvers"
                        title={intl.getMessage('dns_private_reverse_resolvers')}
                        titleClass={cn(theme.title.h4, theme.title.h3_tablet, s.title)}
                        description={
                            <>
                                <p class={s.desc}>
                                    {intl.getMessage('dns_private_reverse_resolvers_desc')}
                                </p>
                                <p class={s.desc}>
                                    {intl.getMessage('dns_private_reverse_resolvers_disabled_desc')}
                                </p>
                            </>
                        }
                        checked={dnsConfigState.use_private_ptr_resolvers}
                        onChange={() => togglePrivatePtrResolvers()}
                        largeTitle
                    />

                    <SettingRow
                        variant="link"
                        id="private_reverse_servers"
                        title={intl.getMessage('dns_private_reverse_servers_title')}
                        description={intl.getMessage('dns_private_reverse_servers_desc')}
                        value={privateReverseValue()}
                        onClick={serversDialog.openDialog}
                    />

                    <SettingRow
                        variant="switch"
                        id="resolve_clients"
                        title={intl.getMessage('dns_private_reverse_resolve_clients_title')}
                        description={intl.getMessage('dns_private_reverse_resolve_clients_desc')}
                        checked={dnsConfigState.resolve_clients}
                        onChange={() => toggleResolveClients()}
                    />
                </div>

                <PrivateReverseServersDialog
                    open={serversDialog.open}
                    onClose={serversDialog.closeDialog}
                    processing={processing()}
                />
            </div>
        </div>
    );
};
