import intl from 'panel/common/intl';

import { PublicHeader } from 'panel/common/ui/PublicHeader';
import { Button } from 'panel/common/ui/Button';
import { Icons } from 'panel/common/ui/Icons';

import s from 'panel/common/ui/Header/Header.module.pcss';

import styles from './styles.module.pcss';

export const ForgotPassword = () => {
    const handleBackToLogin = () => {
        window.location.assign('/login.html');
    };

    return (
        <div class={styles.loginWrapper}>
            <PublicHeader
                dropdownClass={s.dropdown}
                dropdownPosition="bottomRight"
                useLocalLanguage={true}
            />

            <div class={styles.login}>
                <h1 class={styles.titleList}>{intl.getMessage('forgot_password')}</h1>
                <div class={styles.listContainer}>
                    <p class={styles.listDesc}>{intl.getMessage('forgot_password_page_desc')}</p>
                    <p>{intl.getMessage('forgot_password_list_title')}</p>

                    <ol class={styles.list}>
                        <li class={styles.listItem}>
                            {intl.getMessage('forgot_password_list_item_1')}
                        </li>
                        <li class={styles.listItem}>
                            {intl.getMessage('forgot_password_list_item_2')}
                        </li>
                        <li class={styles.listItem}>
                            {intl.getMessage('forgot_password_list_item_3')}
                        </li>
                        <li class={styles.listItem}>
                            {intl.getMessage('forgot_password_list_item_4')}
                        </li>
                        <li class={styles.listItem}>
                            {intl.getMessage('forgot_password_list_item_5')}
                        </li>
                    </ol>

                    <div class={styles.footer}>
                        <Button
                            class={styles.button}
                            id="back_to_login"
                            type="button"
                            variant="primary"
                            size="small"
                            onClick={handleBackToLogin}
                        >
                            {intl.getMessage('back')}
                        </Button>
                    </div>
                </div>
            </div>

            <Icons />
        </div>
    );
};
