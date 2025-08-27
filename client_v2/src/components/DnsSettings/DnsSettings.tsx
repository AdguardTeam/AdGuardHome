import React, { useEffect } from 'react';

import { useDispatch, useSelector } from 'react-redux';

import { getDnsConfig } from 'panel/actions/dnsConfig';
import { getAccessList } from 'panel/actions/access';
import { RootState } from 'panel/initialState';
import intl from 'panel/common/intl';
import cn from 'clsx';

import { PageLoader } from 'panel/common/ui/Loader';
import theme from 'panel/lib/theme';
import { Upstream } from './Upstream';
import { Access } from './Access';
import { ServerConfig } from './ServerConfig';
import { Cache } from './Cache';

export const DnsSettings = () => {
    const dispatch = useDispatch();
    const processing = useSelector((state: RootState) => state.access.processing);
    const processingGetConfig = useSelector((state: RootState) => state.dnsConfig.processingGetConfig);
    const isDataLoading = processing || processingGetConfig;

    useEffect(() => {
        dispatch(getAccessList());
        dispatch(getDnsConfig());
    }, []);

    return (
        <div className={theme.layout.container}>
            <h1 className={cn(theme.layout.title, theme.title.h4, theme.title.h3_tablet)}>
                {intl.getMessage('dns_settings')}
            </h1>

            {isDataLoading ? (
                <PageLoader />
            ) : (
                <>
                    <Upstream />

                    <ServerConfig />

                    <Cache />

                    <Access />
                </>
            )}
        </div>
    );
};
