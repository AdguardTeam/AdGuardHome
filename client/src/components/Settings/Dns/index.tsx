import React, { useEffect } from 'react';
import { useTranslation } from 'react-i18next';

import { useDispatch, useSelector } from '@/store/hooks';




import PageTitle from '@/components/ui/PageTitle';

import Loading from '@/components/ui/Loading';

import { getDnsConfig } from '@/actions/dnsConfig';
import { getAccessList } from '@/actions/access';
import { RootState } from '@/initialState';
import CacheConfig from './Cache';
import Config from './Config';
import Access from './Access';
import Upstream from './Upstream';

const Dns = () => {
    const { t } = useTranslation();
    const dispatch = useDispatch();

    const processing = useSelector((state: RootState) => state.access.processing);

    const processingGetConfig = useSelector((state: RootState) => state.dnsConfig.processingGetConfig);

    const isDataLoading = processing || processingGetConfig;

    useEffect(() => {
        dispatch(getAccessList());
        dispatch(getDnsConfig());
    }, []);

    return (
        <>
            <PageTitle title={t('dns_settings')} />
            {isDataLoading ? (
                <Loading />
            ) : (
                <>
                    <Upstream />

                    <Config />

                    <CacheConfig />

                    <Access />
                </>
            )}
        </>
    );
};

export default Dns;
