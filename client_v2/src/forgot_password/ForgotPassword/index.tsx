import intl from 'panel/common/intl';

import theme from 'panel/lib/theme';
import { AuthLayout } from 'panel/common/ui/AuthLayout';
import { Button } from 'panel/common/ui/Button';

import styles from './styles.module.pcss';

export const ForgotPassword = () => {
    const handleBackToLogin = () => {
        window.location.assign('/login.html');
    };

    return (
        <AuthLayout title={intl.getMessage('forgot_password')}>
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
            </div>

            <div class={theme.auth.footer}>
                <div class={theme.auth.footerRow}>
                    <Button
                        class={theme.auth.footerButton}
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
        </AuthLayout>
    );
};
