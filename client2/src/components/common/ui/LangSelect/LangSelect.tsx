import React, { FC, useContext } from 'react';

import { Icon } from 'Common/ui';
import Store from 'Store';
import { LANGUAGES } from 'Localization';

import s from './LangSelect.module.pcss';

const LangSelector: FC = () => {
    const store = useContext(Store);
    const { ui: { currentLang } } = store;

    const lang = LANGUAGES.find((e) => e.code === currentLang)!;

    return (
        <div className={s.wrap}>
            <Icon icon="language" className={s.icon} />
            {lang.name}
        </div>
    );
};

export default LangSelector;
