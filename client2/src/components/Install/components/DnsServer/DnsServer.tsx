import React, { FC, useContext } from 'react';
import cn from 'classnames';
import { observer } from 'mobx-react-lite';
import { FormikHelpers } from 'formik';

import { Input, Radio, Switch } from 'Common/controls';
import { DEFAULT_IP_ADDRESS } from 'Consts/install';
import { chechNetworkType, NETWORK_TYPE } from 'Helpers/installHelpers';
import theme from 'Lib/theme';
import Store from 'Store/installStore';

import s from './DnsServer.module.pcss';
import { FormValues } from '../../Install';
import StepButtons from '../StepButtons';

enum NETWORK_OPTIONS {
    ALL = 'all',
    CUSTOM = 'custom',
}

interface DnsServerProps {
    values: FormValues;
    setFieldValue: FormikHelpers<FormValues>['setFieldValue'];
}

const DnsServer: FC<DnsServerProps> = observer(({
    values,
    setFieldValue,
}) => {
    const { ui: { intl }, install: { addresses } } = useContext(Store);
    const { dns: { ip } } = values;
    const radioValue = ip.length === 1 && ip[0] === DEFAULT_IP_ADDRESS
        ? NETWORK_OPTIONS.ALL : NETWORK_OPTIONS.CUSTOM;

    const onSelectRadio = (v: string | number) => {
        const value = v === NETWORK_OPTIONS.ALL
            ? [DEFAULT_IP_ADDRESS] : [];
        setFieldValue('dns.ip', value);
    };

    const getManualBlock = () => (
        <div className={s.manualOptions}>
            {addresses?.interfaces.map((a) => {
                let name = '';
                const type = chechNetworkType(a.name);
                switch (type) {
                    case NETWORK_TYPE.ETHERNET:
                        name = `${intl.getMessage('ethernet')} (${a.name}) `;
                        break;
                    case NETWORK_TYPE.LOCAL:
                        name = `${intl.getMessage('localhost')} (${a.name}) `;
                        break;
                    default:
                        name = a.name || '';
                        break;
                }
                return (
                    <div key={a.name}>
                        <div>
                            <div className={s.name}>
                                {name}
                            </div>
                            {a.ipAddresses?.map((addrIp) => (
                                <div key={addrIp} className={s.manualOption}>
                                    <div className={theme.typography.subtext}>
                                        {addrIp}
                                    </div>
                                    <Switch
                                        checked={values.dns.ip.includes(addrIp)}
                                        onChange={() => {
                                            const temp = new Set(ip);
                                            if (temp.has(addrIp)) {
                                                temp.delete(addrIp);
                                            } else {
                                                temp.add(addrIp);
                                            }
                                            setFieldValue('dns.ip', Array.from(temp.values()));
                                        }}/>
                                </div>
                            ))}
                        </div>
                    </div>
                );
            })}
        </div>
    );

    return (
        <div>
            <div className={theme.typography.title}>
                {intl.getMessage('install_dns_server_title')}
            </div>
            <div className={cn(theme.typography.text, theme.typography.text_block)}>
                {intl.getMessage('install_dns_server_desc')}
            </div>
            <div className={theme.typography.subTitle}>
                {intl.getMessage('install_dns_server_network_interfaces')}
            </div>
            <div className={cn(theme.typography.text, theme.typography.text_base)}>
                {intl.getMessage('install_dns_server_network_interfaces_desc')}
            </div>
            <Radio
                value={radioValue}
                onSelect={onSelectRadio}
                options={[
                    {
                        value: NETWORK_OPTIONS.ALL,
                        label: intl.getMessage('install_all_networks'),
                        desc: intl.getMessage('install_all_networks_description'),
                    },
                    {
                        value: NETWORK_OPTIONS.CUSTOM,
                        label: intl.getMessage('install_choose_networks'),
                        desc: intl.getMessage('install_choose_networks_desc'),
                    },
                ]}
            />
            { radioValue !== NETWORK_OPTIONS.ALL && getManualBlock()}
            <div className={theme.typography.subTitle}>
                {intl.getMessage('install_dns_server_port')}
            </div>
            <div className={cn(theme.typography.text, theme.typography.text_base)}>
                {intl.getMessage('install_dns_server_port_desc')}
            </div>
            <Input
                label={`${intl.getMessage('port')}:`}
                placeholder={intl.getMessage('port')}
                type="number"
                name="dnsPort"
                value={values.dns.port}
                onChange={(v) => {
                    const port = v === '' ? '' : parseInt(v, 10);
                    setFieldValue('dns.port', port);
                }}
            />
            <StepButtons
                setFieldValue={setFieldValue}
                currentStep={3}
                values={values}
            />
        </div>
    );
});

export default DnsServer;
