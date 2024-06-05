import React, { useEffect } from 'react';
import { useTranslation } from 'react-i18next';

import { useDispatch, useSelector } from 'react-redux';

import Upstream from './Upstream';

import Access from './Access';

import Config from './Config';

import PageTitle from '../../ui/PageTitle';

import Loading from '../../ui/Loading';

import CacheConfig from './Cache';
import { getDnsConfig } from '../../../actions/dnsConfig';
import { getAccessList } from '../../../actions/access';
import { RootState } from '../../../initialState';

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
