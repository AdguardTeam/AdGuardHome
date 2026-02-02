import React from 'react';

import intl from 'panel/common/intl';

import { PublicHeader } from 'panel/common/ui/PublicHeader';
import { Button } from 'panel/common/ui/Button';
import { Icons } from 'panel/common/ui/Icons';

import s from 'panel/common/ui/Header/Header.module.pcss';
import twosky from '../../../../.twosky.json';

import styles from './styles.module.pcss';

const LANGUAGES = twosky[1].languages;

export const ForgotPassword = () => {
    const handleBackToLogin = () => {
        window.location.assign('/login.html');
    };

    return (
        <div className={styles.loginWrapper}>
            <PublicHeader
                languages={LANGUAGES}
                dropdownClassName={s.dropdown}
                dropdownPosition="bottomRight"
            />

            <div className={styles.login}>
                <h1 className={styles.titleList}>{intl.getMessage('forgot_password')}</h1>
                <div className={styles.listContainer}>
                    <p className={styles.listDesc}>{intl.getMessage('forgot_password_page_desc')}</p>
                    <p>{intl.getMessage('forgot_password_list_title')}</p>

                    <ol className={styles.list}>
                        <li className={styles.listItem}>{intl.getMessage('forgot_password_list_item_1')}</li>
                        <li className={styles.listItem}>{intl.getMessage('forgot_password_list_item_2')}</li>
                        <li className={styles.listItem}>{intl.getMessage('forgot_password_list_item_3')}</li>
                        <li className={styles.listItem}>{intl.getMessage('forgot_password_list_item_4')}</li>
                        <li className={styles.listItem}>{intl.getMessage('forgot_password_list_item_5')}</li>
                    </ol>

                    <div className={styles.footer}>
                        <Button
                            className={styles.button}
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
