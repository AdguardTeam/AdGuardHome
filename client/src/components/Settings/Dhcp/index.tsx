import React, { useEffect, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import { shallowEqual, useDispatch, useSelector } from 'react-redux';
import classNames from 'classnames';

import { FormProvider, useForm } from 'react-hook-form';
import { DHCP_DESCRIPTION_PLACEHOLDERS, STATUS_RESPONSE } from '../../../helpers/constants';

import Leases from './Leases';

import StaticLeases from './StaticLeases/index';

import Card from '../../ui/Card';

import PageTitle from '../../ui/PageTitle';

import Loading from '../../ui/Loading';
import {
    findActiveDhcp,
    getDhcpInterfaces,
    getDhcpStatus,
    resetDhcp,
    setDhcpConfig,
    resetDhcpLeases,
    toggleDhcp,
    toggleLeaseModal,
} from '../../../actions';

import FormDHCPv4 from './FormDHCPv4';

import FormDHCPv6 from './FormDHCPv6';

import Interfaces from './Interfaces';
import {
    calculateDhcpPlaceholdersIpv4,
    calculateDhcpPlaceholdersIpv6,
    subnetMaskToBitMask,
} from '../../../helpers/helpers';
import './index.css';
import { RootState } from '../../../initialState';

type IPv4FormValues = {
    gateway_ip?: string;
    subnet_mask?: string;
    range_start?: string;
    range_end?: string;
    lease_duration?: number;
}

type IPv6FormValues = {
    range_start?: string;
    range_end?: string;
    lease_duration?: number;
}

const getDefaultV4Values = (v4: IPv4FormValues) => {
    const emptyForm = Object.entries(v4).every(
        ([key, value]) => key === 'lease_duration' || value === ''
    );

    if (emptyForm) {
        return {
            ...v4,
            lease_duration: undefined,
        }
    }

    return v4;
}

export type DhcpFormValues = {
    v4?: IPv4FormValues;
    v6?: IPv6FormValues;
    interface_name?: string;
};

const DEFAULT_V4_VALUES = {
    gateway_ip: '',
    subnet_mask: '',
    range_start: '',
    range_end: '',
    lease_duration: undefined,
};

const DEFAULT_V6_VALUES = {
    range_start: '',
    range_end: '',
    lease_duration: undefined,
};

const Dhcp = () => {
    const { t } = useTranslation();
    const dispatch = useDispatch();
    const {
        processingStatus,
        processingConfig,
        processing,
        processingInterfaces,
        check,
        leases,
        staticLeases,
        isModalOpen,
        processingAdding,
        processingDeleting,
        processingUpdating,
        processingDhcp,
        v4,
        v6,
        interface_name: interfaceName,
        enabled,
        dhcp_available,
        interfaces,
        modalType,
    } = useSelector((state: RootState) => state.dhcp, shallowEqual);

    const methods = useForm<DhcpFormValues>({
        mode: 'onBlur',
        defaultValues: {
            v4: getDefaultV4Values(v4),
            v6,
            interface_name: interfaceName || '',
        },
    });
    const { watch, reset } = methods;

    const interface_name = watch('interface_name');
    const isInterfaceIncludesIpv4 = useSelector(
        (state: RootState) => !!state.dhcp?.interfaces?.[interface_name]?.ipv4_addresses,
    );
    const ipv4Config = watch('v4');

    const [ipv4placeholders, setIpv4Placeholders] = useState(DHCP_DESCRIPTION_PLACEHOLDERS.ipv4);
    const [ipv6placeholders, setIpv6Placeholders] = useState(DHCP_DESCRIPTION_PLACEHOLDERS.ipv6);

    useEffect(() => {
        dispatch(getDhcpStatus());
    }, []);

    useEffect(() => {
        if (dhcp_available) {
            dispatch(getDhcpInterfaces());
        }
    }, [dhcp_available]);

    useEffect(() => {
        if (v4 || v6 || interfaceName) {
            reset({
                v4: {
                    ...DEFAULT_V4_VALUES,
                    ...getDefaultV4Values(v4),
                },
                v6: {
                    ...DEFAULT_V6_VALUES,
                    ...v6,
                },
                interface_name: interfaceName || '',
            });
        }
    }, [v4, v6, interfaceName, reset]);

    useEffect(() => {
        const [ipv4] = interfaces?.[interface_name]?.ipv4_addresses ?? [];
        const [ipv6] = interfaces?.[interface_name]?.ipv6_addresses ?? [];
        const gateway_ip = interfaces?.[interface_name]?.gateway_ip;

        const v4placeholders = ipv4
            ? calculateDhcpPlaceholdersIpv4(ipv4, gateway_ip)
            : DHCP_DESCRIPTION_PLACEHOLDERS.ipv4;

        const v6placeholders = ipv6 ? calculateDhcpPlaceholdersIpv6() : DHCP_DESCRIPTION_PLACEHOLDERS.ipv6;

        setIpv4Placeholders(v4placeholders);
        setIpv6Placeholders(v6placeholders);
    }, [interface_name]);

    const clear = () => {
        // eslint-disable-next-line no-alert
        if (window.confirm(t('dhcp_reset'))) {
            reset({
                v4: DEFAULT_V4_VALUES,
                v6: DEFAULT_V6_VALUES,
                interface_name: '',
            });
            dispatch(resetDhcp());
            dispatch(getDhcpStatus());
        }
    };

    const handleSubmit = (values: DhcpFormValues) => {
        dispatch(
            setDhcpConfig({
                interface_name,
                ...values,
            }),
        );
    };

    const handleReset = () => {
        if (window.confirm(t('dhcp_reset_leases_confirm'))) {
            dispatch(resetDhcpLeases());
        }
    };

    const enteredSomeV4Value = Object.values(v4).some(Boolean);

    const enteredSomeV6Value = Object.values(v6).some(Boolean);
    const enteredSomeValue = enteredSomeV4Value || enteredSomeV6Value || interfaceName;

    const getToggleDhcpButton = () => {
        const filledConfig = interface_name && (Object.values(v4).every(Boolean) || Object.values(v6).every(Boolean));

        const className = classNames('btn btn-sm', {
            'btn-gray': enabled,
            'btn-outline-success': !enabled,
        });

        const onClickDisable = () => dispatch(toggleDhcp({ enabled }));
        const onClickEnable = () => {
            const values = {
                enabled,
                interface_name,
                v4: enteredSomeV4Value ? v4 : {},
                v6: enteredSomeV6Value ? v6 : {},
            };
            dispatch(toggleDhcp(values));
        };

        return (
            <button
                type="button"
                className={className}
                onClick={enabled ? onClickDisable : onClickEnable}
                disabled={processingDhcp || processingConfig || (!enabled && (!filledConfig || !check))}>
                <Trans>{enabled ? 'dhcp_disable' : 'dhcp_enable'}</Trans>
            </button>
        );
    };

    const statusButtonClass = classNames('btn btn-sm dhcp-form__button', {
        'btn-loading btn-primary': processingStatus,
        'btn-outline-primary': !processingStatus,
    });

    const onClick = () => dispatch(findActiveDhcp(interface_name));

    const toggleModal = () => dispatch(toggleLeaseModal());

    if (processing || processingInterfaces) {
        return <Loading />;
    }

    if (!processing && !dhcp_available) {
        return (
            <div className="text-center pt-5">
                <h2>
                    <Trans>unavailable_dhcp</Trans>
                </h2>

                <h4>
                    <Trans>unavailable_dhcp_desc</Trans>
                </h4>
            </div>
        );
    }

    const toggleDhcpButton = getToggleDhcpButton();

    const inputtedIPv4values = ipv4Config.gateway_ip && ipv4Config.subnet_mask;

    const isEmptyConfig = !Object.values(ipv4Config).some(Boolean);
    const disabledLeasesButton = Boolean(
        !isInterfaceIncludesIpv4 || isEmptyConfig || processingConfig || !inputtedIPv4values,
    );
    const cidr = inputtedIPv4values ? `${ipv4Config.gateway_ip}/${subnetMaskToBitMask(ipv4Config.subnet_mask)}` : '';

    return (
        <>
            <PageTitle title={t('dhcp_settings')} subtitle={t('dhcp_description')} containerClass="page-title--dhcp">
                {toggleDhcpButton}

                <button
                    type="button"
                    className={statusButtonClass}
                    onClick={onClick}
                    disabled={enabled || !interface_name || processingConfig}>
                    <Trans>check_dhcp_servers</Trans>
                </button>

                <button
                    type="button"
                    className="btn btn-sm btn-outline-secondary"
                    disabled={!enteredSomeValue || processingConfig}
                    onClick={clear}>
                    <Trans>reset_settings</Trans>
                </button>
            </PageTitle>
            {!processing && !processingInterfaces && (
                <>
                    {!enabled &&
                        check &&
                        (check.v4.other_server.found !== STATUS_RESPONSE.NO ||
                            check.v6.other_server.found !== STATUS_RESPONSE.NO) && (
                            <div className="mb-5">
                                <hr />

                                <div className="text-danger">
                                    <Trans>dhcp_warning</Trans>
                                </div>
                            </div>
                        )}

                    <FormProvider {...methods}>
                        <Interfaces />
                        <Card title={t('dhcp_ipv4_settings')} bodyType="card-body box-body--settings">
                            <div>
                                <FormDHCPv4
                                    onSubmit={handleSubmit}
                                    processingConfig={processingConfig}
                                    ipv4placeholders={ipv4placeholders}
                                    interfaces={interfaces}
                                />
                            </div>
                        </Card>
                        <Card title={t('dhcp_ipv6_settings')} bodyType="card-body box-body--settings">
                            <div>
                                <FormDHCPv6
                                    onSubmit={handleSubmit}
                                    processingConfig={processingConfig}
                                    ipv6placeholders={ipv6placeholders}
                                    interfaces={interfaces}
                                />
                            </div>
                        </Card>
                    </FormProvider>

                    {enabled && (
                        <Card title={t('dhcp_leases')} bodyType="card-body box-body--settings">
                            <div className="row">
                                <div className="col">
                                    <Leases leases={leases} disabledLeasesButton={disabledLeasesButton} />
                                </div>
                            </div>
                        </Card>
                    )}

                    <Card title={t('dhcp_static_leases')} bodyType="card-body box-body--settings">
                        <div className="row">
                            <div className="col-12">
                                <StaticLeases
                                    staticLeases={staticLeases}
                                    isModalOpen={isModalOpen}
                                    modalType={modalType}
                                    processingAdding={processingAdding}
                                    processingDeleting={processingDeleting}
                                    processingUpdating={processingUpdating}
                                    cidr={cidr}
                                    gatewayIp={ipv4Config.gateway_ip}
                                />

                                <div className="btn-list mt-2">
                                    <button
                                        type="button"
                                        className="btn btn-success btn-standard mt-3"
                                        onClick={toggleModal}
                                        disabled={disabledLeasesButton}>
                                        <Trans>dhcp_add_static_lease</Trans>
                                    </button>

                                    <button
                                        type="button"
                                        className="btn btn-secondary btn-standard mt-3"
                                        onClick={handleReset}>
                                        <Trans>dhcp_reset_leases</Trans>
                                    </button>
                                </div>
                            </div>
                        </div>
                    </Card>
                </>
            )}
        </>
    );
};

export default Dhcp;
