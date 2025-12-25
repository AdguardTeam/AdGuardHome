import React, { useMemo } from 'react';
import { withTranslation } from 'react-i18next';

import intl from 'panel/common/intl';
import Controls from './Controls';

const Greeting = () => {
    const configureList = useMemo(
        () => [
            intl.getMessage('setup_guide_greeting_list_1'),
            intl.getMessage('setup_guide_greeting_list_2'),
            intl.getMessage('setup_guide_greeting_list_3'),
            intl.getMessage('setup_guide_greeting_list_4'),
            intl.getMessage('setup_guide_greeting_list_5'),
        ],
        [],
    );

    return (
        <div className="setup__greeting">
            <div className="setup__left-side">
                <h1 className="setup__title">{intl.getMessage('setup_guide_greeting_title')}</h1>

                <p className="setup__desc">{intl.getMessage('setup_guide_greeting_desc')}</p>

                <ul className="setup__list">
                    {configureList.map((item, idx) => (
                        <li key={idx} className="setup__item">
                            {item}
                        </li>
                    ))}
                </ul>

                <Controls />
            </div>

            <div className="setup__right-side">
                <img src="https://cdn.adguardcdn.com/website/adguard.com/products/screenshots/home/adguard_home.svg?_plc=ru" alt="" />
            </div>
        </div>
    );
};

export default withTranslation()(Greeting);

