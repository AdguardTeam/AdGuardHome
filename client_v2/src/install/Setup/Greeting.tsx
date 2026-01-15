import React, { useMemo } from 'react';

import intl from 'panel/common/intl';
import Controls from './Controls';
import setup from './styles.module.pcss'

import routerImage from '../../img/router.svg';

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
        <div className={setup.greeting}>
            <div className={setup.info}>
                <h1 className={setup.title}>{intl.getMessage('setup_guide_greeting_title')}</h1>

                <p className={setup.desc}>{intl.getMessage('setup_guide_greeting_desc')}</p>

                <ul className={setup.list}>
                    {configureList.map((item, idx) => (
                        <li key={idx} className={setup.listItem}>
                            {item}
                        </li>
                    ))}
                </ul>

                <Controls />
            </div>

            <div className={setup.content}>
                <img src={routerImage} alt="Router" />
            </div>
        </div>
    );
};

export default Greeting;

