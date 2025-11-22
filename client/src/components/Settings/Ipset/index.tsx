import React, { useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { useDispatch, useSelector } from 'react-redux';

import PageTitle from '../../ui/PageTitle';
import Loading from '../../ui/Loading';
import Form from './Form';

import { getDnsConfig, setDnsConfig } from '../../../actions/dnsConfig';
import { RootState } from '../../../initialState';

const Ipset: React.FC = () => {
    const { t } = useTranslation();
    const dispatch = useDispatch();

    const processingGetConfig = useSelector((state: RootState) => state.dnsConfig.processingGetConfig);
    const processingSetConfig = useSelector((state: RootState) => state.dnsConfig.processingSetConfig);
    const ipset = useSelector((state: RootState) => state.dnsConfig.ipset || []);
    const ipset_file = useSelector((state: RootState) => state.dnsConfig.ipset_file || '');
    const ipset_create = useSelector((state: RootState) => state.dnsConfig.ipset_create || null);

    useEffect(() => {
        dispatch(getDnsConfig());
    }, [dispatch]);

    const handleSubmit = (data: { ipset: string[]; ipset_file: string; ipset_create: any }) => {
        dispatch(setDnsConfig(data));
    };

    if (processingGetConfig) {
        return (
            <>
                <PageTitle title={t('ipset_title')} />
                <Loading />
            </>
        );
    }

    return (
        <>
            <PageTitle title={t('ipset_title')} />
            <div className="content">
                <div className="container">
                    <Form
                        initialRules={ipset}
                        initialFilePath={ipset_file}
                        initialIpsetCreate={ipset_create}
                        onSubmit={handleSubmit}
                        processing={processingSetConfig}
                    />
                </div>
            </div>
        </>
    );
};

export default Ipset;
