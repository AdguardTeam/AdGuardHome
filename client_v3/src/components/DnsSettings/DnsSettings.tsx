import { createMemo, Show, onMount } from 'solid-js';

import { dnsConfigState, getDnsConfig } from 'panel/stores/dnsConfig';
import { accessState, getAccessList } from 'panel/stores/access';
import intl from 'panel/common/intl';
import cn from 'clsx';

import { PageLoader } from 'panel/common/ui/Loader';
import theme from 'panel/lib/theme';
import { Upstream } from './Upstream';
import { Access } from './Access';
import { ServerConfig } from './ServerConfig';
import { Cache } from './Cache';

export const DnsSettings = () => {
    // upstream_dns is only defined after a successful fetch.
    const hasCachedData = createMemo(() => dnsConfigState.upstream_dns !== undefined);
    const isDataLoading = createMemo(
        () => !hasCachedData() && (accessState.processing || dnsConfigState.processingGetConfig),
    );

    onMount(() => {
        getAccessList();
        getDnsConfig();
    });

    return (
        <div class={theme.layout.container}>
            <div class={cn(theme.layout.containerIn, theme.layout.containerIn_one_col)}>
                <h1 class={cn(theme.layout.title, theme.title.h4, theme.title.h3_tablet)}>
                    {intl.getMessage('dns_settings')}
                </h1>

                <Show
                    when={isDataLoading()}
                    fallback={
                        <>
                            <Upstream />
                            <ServerConfig />
                            <Cache />
                            <Access />
                        </>
                    }
                >
                    <PageLoader />
                </Show>
            </div>
        </div>
    );
};
