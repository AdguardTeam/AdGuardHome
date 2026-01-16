import React, { useMemo } from 'react';

import intl from 'panel/common/intl';
import Controls from './Controls';
import styles from './styles.module.pcss';

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
        <div className={styles.greeting}>
            <div className={styles.info}>
                <h1 className={styles.title}>{intl.getMessage('setup_guide_greeting_title')}</h1>

                <p className={styles.desc}>{intl.getMessage('setup_guide_greeting_desc')}</p>

                <ul className={styles.list}>
                    {configureList.map((item, idx) => (
                        <li key={idx} className={styles.listItem}>
                            {item}
                        </li>
                    ))}
                </ul>

                <Controls />
            </div>

            <div className={styles.content}>
                <img src={routerImage} alt="Router" />
            </div>
        </div>
    );
};

export default Greeting;

