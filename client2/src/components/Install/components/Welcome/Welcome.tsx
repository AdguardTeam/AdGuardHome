import React, { FC, useContext } from 'react';
import { Button } from 'antd';
import { observer } from 'mobx-react-lite';

import Store from 'Store/installStore';
import Icon from 'Common/ui/Icon';
import theme from 'Lib/theme';

import s from './Welcome.module.pcss';

interface WelcomeProps {
    onNext: () => void;
}

const Welcome: FC<WelcomeProps> = observer(({ onNext }) => {
    const { ui: { intl } } = useContext(Store);
    return (
        <div className={s.content}>
            <div className={s.iconContainer}>
                <Icon icon="mainLogo" className={s.icon} />
            </div>
            <div className={theme.typography.title}>
                {intl.getMessage('install_wellcome_title')}
            </div>
            <div className={theme.typography.text}>
                {intl.getMessage('install_wellcome_desc')}
            </div>
            <Button
                size="large"
                type="primary"
                className={s.button}
                onClick={onNext}
            >
                {intl.getMessage('install_wellcome_button')}
            </Button>
        </div>
    );
});

export default Welcome;
