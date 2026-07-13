import { createMemo } from 'solid-js';
import cn from 'clsx';

import { accessState } from 'panel/stores/access';
import intl from 'panel/common/intl';
import { SettingRow } from 'panel/common/ui/SettingRow';
import { useDialog } from 'panel/hooks/useDialog';
import { getListSummary } from '../helpers';
import theme from 'panel/lib/theme';

import { AllowedClientsDialog } from './blocks/AllowedClientsDialog';
import { DisallowedClientsDialog } from './blocks/DisallowedClientsDialog';
import { DisallowedDomainsDialog } from './blocks/DisallowedDomainsDialog';

export const Access = () => {
    const allowedDialog = useDialog();
    const disallowedClientsDialog = useDialog();
    const disallowedDomainsDialog = useDialog();

    const allowedClientsOn = createMemo(() => accessState.allowed_clients.trim().length > 0);
    const processing = () => accessState.processingSet;

    const allowedClientsValue = createMemo(() => getListSummary(accessState.allowed_clients));
    const disallowedClientsValue = createMemo(() => getListSummary(accessState.disallowed_clients));
    const disallowedDomainsValue = createMemo(() => getListSummary(accessState.blocked_hosts));

    return (
        <div>
            <h2 class={cn(theme.layout.subtitle, theme.title.h5, theme.title.h4_tablet)}>
                {intl.getMessage('dns_access_settings_title')}
            </h2>

            <SettingRow
                variant="link"
                id="allowed_clients"
                title={intl.getMessage('dns_allowed_clients')}
                description={intl.getMessage('dns_allowed_clients_desc')}
                value={allowedClientsValue()}
                onClick={allowedDialog.openDialog}
            />

            <SettingRow
                variant="link"
                id="disallowed_clients"
                title={intl.getMessage('dns_disallowed_clients')}
                description={intl.getMessage('dns_disallowed_clients_desc')}
                value={disallowedClientsValue()}
                disabled={allowedClientsOn()}
                onClick={disallowedClientsDialog.openDialog}
            />

            <SettingRow
                variant="link"
                id="disallowed_domains"
                title={intl.getMessage('dns_disallowed_domains')}
                description={intl.getMessage('dns_disallowed_domains_desc')}
                value={disallowedDomainsValue()}
                onClick={disallowedDomainsDialog.openDialog}
            />

            <AllowedClientsDialog
                open={allowedDialog.open}
                onClose={allowedDialog.closeDialog}
                processing={processing()}
            />

            <DisallowedClientsDialog
                open={disallowedClientsDialog.open}
                onClose={disallowedClientsDialog.closeDialog}
                processing={processing()}
            />

            <DisallowedDomainsDialog
                open={disallowedDomainsDialog.open}
                onClose={disallowedDomainsDialog.closeDialog}
                processing={processing()}
            />
        </div>
    );
};
