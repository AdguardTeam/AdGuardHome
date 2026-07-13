import intl from 'panel/common/intl';

import { PublicHeader } from 'panel/common/ui/PublicHeader';
import { Icons } from 'panel/common/ui/Icons';

import s from 'panel/common/ui/Header/Header.module.pcss';
import { processLogin } from 'panel/stores/login';
import Toasts from '../../components/Toasts';
import Form, { type LoginFormValues } from './Form';
import styles from './styles.module.pcss';

export const Login = () => {
    const handleSubmit = (values: LoginFormValues) => {
        processLogin({ name: values.username, password: values.password });
    };

    return (
        <div class={styles.loginWrapper}>
            <PublicHeader
                dropdownClass={s.dropdown}
                dropdownPosition="bottomRight"
                useLocalLanguage={true}
            />
            <div class={styles.login}>
                <h1 class={styles.title}>{intl.getMessage('login')}</h1>
                <Form onSubmit={handleSubmit} />
            </div>

            <Toasts />

            <Icons />
        </div>
    );
};
