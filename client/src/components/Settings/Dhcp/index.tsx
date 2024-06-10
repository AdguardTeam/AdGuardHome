import React, { useEffect, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import { shallowEqual, useDispatch, useSelector } from 'react-redux';
import classNames from 'classnames';

import { destroy } from 'redux-form';
import { DHCP_DESCRIPTION_PLACEHOLDERS, DHCP_FORM_NAMES, STATUS_RESPONSE, FORM_NAME } from '../../../helpers/constants';

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

    const interface_name =
        useSelector((state: RootState) => state.form[FORM_NAME.DHCP_INTERFACES]?.values?.interface_name);
    const isInterfaceIncludesIpv4 =
        useSelector((state: RootState) => !!state.dhcp?.interfaces?.[interface_name]?.ipv4_addresses);

    const dhcp = useSelector((state: RootState) => state.form[FORM_NAME.DHCPv4], shallowEqual);

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
            Object.values(DHCP_FORM_NAMES).forEach((formName: any) => dispatch(destroy(formName)));
            dispatch(resetDhcp());
            dispatch(getDhcpStatus());
        }
    };

    const handleSubmit = (values: any) => {
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
        const filledConfig =
            interface_name &&
            (Object.values(v4)

                .every(Boolean) ||
                Object.values(v6).every(Boolean));

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

    const initialV4 = enteredSomeV4Value ? v4 : {};
    const initialV6 = enteredSomeV6Value ? v6 : {};

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

    const inputtedIPv4values = dhcp?.values?.v4?.gateway_ip && dhcp?.values?.v4?.subnet_mask;

    const isEmptyConfig = !Object.values(dhcp?.values?.v4 ?? {}).some(Boolean);
    const disabledLeasesButton = Boolean(
        dhcp?.syncErrors ||
            !isInterfaceIncludesIpv4 ||
            isEmptyConfig ||
            processingConfig ||
            !inputtedIPv4values,
    );
    const cidr = inputtedIPv4values
        ? `${dhcp?.values?.v4?.gateway_ip}/${subnetMaskToBitMask(dhcp?.values?.v4?.subnet_mask)}`
        : '';

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

                    <Interfaces initialValues={{ interface_name: interfaceName }} />

                    <Card title={t('dhcp_ipv4_settings')} bodyType="card-body box-body--settings">
                        <div>
                            <FormDHCPv4
                                onSubmit={handleSubmit}
                                initialValues={{ v4: initialV4 }}
                                processingConfig={processingConfig}
                                ipv4placeholders={ipv4placeholders}
                            />
                        </div>
                    </Card>

                    <Card title={t('dhcp_ipv6_settings')} bodyType="card-body box-body--settings">
                        <div>
                            <FormDHCPv6
                                onSubmit={handleSubmit}
                                initialValues={{ v6: initialV6 }}
                                processingConfig={processingConfig}
                                ipv6placeholders={ipv6placeholders}
                            />
                        </div>
                    </Card>
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
                                    gatewayIp={dhcp?.values?.v4?.gateway_ip}
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
