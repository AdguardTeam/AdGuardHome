import React, { FC, useContext } from 'react';
import { Button } from 'antd';
import { observer } from 'mobx-react-lite';

import Store from 'Store/installStore';
import Icon from 'Common/ui/Icon';
import theme from 'Lib/theme';

interface WelcomeProps {
    onNext: () => void;
}

const Welcome: FC<WelcomeProps> = observer(({ onNext }) => {
    const { ui: { intl } } = useContext(Store);
    return (
        <>
            <Icon icon="logo" className={theme.install.logo} />
            <div className={theme.install.title}>
                {intl.getMessage('install_wellcome_title')}
            </div>
            <div className={theme.install.text}>
                {intl.getMessage('install_wellcome_desc')}
            </div>
            <div className={theme.install.actions}>
                <Button
                    size="large"
                    type="primary"
                    className={theme.install.button}
                    onClick={onNext}
                >
                    {intl.getMessage('install_wellcome_button')}
                </Button>
            </div>
        </>
    );
});

export default Welcome;
