import React, { FC } from 'react';
import { Layout } from 'antd';
import { Formik, FormikHelpers } from 'formik';
import { observer } from 'mobx-react-lite';
import cn from 'classnames';

import { IInitialConfigurationBeta } from 'Entities/InitialConfigurationBeta';
import Icons from 'Common/ui/Icons';
import {
    DEFAULT_DNS_ADDRESS,
    DEFAULT_DNS_PORT,
    DEFAULT_IP_ADDRESS,
    DEFAULT_IP_PORT,
} from 'Consts/install';
import { notifyError } from 'Common/ui';
import InstallStore from 'Store/stores/Install';
import theme from 'Lib/theme';

import AdminInterface from './components/AdminInterface';
import Auth from './components/Auth';
import DnsServer from './components/DnsServer';
import Stepper from './components/Stepper';
import Welcome from './components/Welcome';
import ConfigureDevices from './components/ConfigureDevices';

const { Content } = Layout;

export type FormValues = IInitialConfigurationBeta & { step: number };

const InstallForm: FC = observer(() => {
    const initialValues: FormValues = {
        step: 0,
        web: {
            ip: [DEFAULT_IP_ADDRESS],
            port: DEFAULT_IP_PORT,
        },
        dns: {
            ip: [DEFAULT_DNS_ADDRESS],
            port: DEFAULT_DNS_PORT,
        },
        password: '',
        username: '',
    };

    const onNext = async (values: FormValues, { setFieldValue }: FormikHelpers<FormValues>) => {
        const currentStep = values.step;
        const checker = (condition: boolean, message: string) => {
            if (condition) {
                setFieldValue('step', currentStep + 1);
            } else {
                notifyError(message);
            }
        };
        switch (currentStep) {
            case 1: {
                // web
                const check = await InstallStore.checkConfig(values);
                checker(check?.web?.status === '', check?.web?.status || '');
                break;
            }
            case 3: {
                // dns
                const check = await InstallStore.checkConfig(values);
                checker(check?.dns?.status === '', check?.dns?.status || '');
                break;
            }
            case 4: {
                // configure
                const config = await InstallStore.configure(values);
                if (config) {
                    const { web } = values;
                    window.location.href = `http://${web.ip[0]}:${web.port}`;
                }
                break;
            }
            default:
                setFieldValue('step', currentStep + 1);
                break;
        }
    };

    return (
        <Formik
            initialValues={initialValues}
            onSubmit={onNext}
        >
            {({ values, handleSubmit, setFieldValue }) => (
                <form noValidate onSubmit={handleSubmit}>
                    <Stepper currentStep={values.step} />
                    {values.step === 0 && (
                        <Welcome onNext={() => setFieldValue('step', 1)}/>
                    )}
                    {values.step === 1 && (
                        <AdminInterface values={values} setFieldValue={setFieldValue} />
                    )}
                    {values.step === 2 && (
                        <Auth values={values} setFieldValue={setFieldValue} />
                    )}
                    {values.step === 3 && (
                        <DnsServer values={values} setFieldValue={setFieldValue} />
                    )}
                    {values.step === 4 && (
                        <ConfigureDevices values={values} setFieldValue={setFieldValue} />
                    )}
                </form>
            )}
        </Formik>
    );
});

const Install: FC = () => {
    return (
        <Layout className={cn(theme.content.content, theme.content.content_auth)}>
            <Content className={cn(theme.content.container, theme.content.container_auth)}>
                <InstallForm />
            </Content>
            <Icons/>
        </Layout>
    );
};

export default Install;
