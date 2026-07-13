import { For, createMemo } from 'solid-js';

import intl from 'panel/common/intl';
import { Controls } from './Controls';
import styles from './styles.module.pcss';

import routerImage from '../../img/router.svg';

const Greeting = () => {
    const configureList = createMemo(() => [
        intl.getMessage('setup_guide_greeting_list_1'),
        intl.getMessage('setup_guide_greeting_list_2'),
        intl.getMessage('setup_guide_greeting_list_3'),
        intl.getMessage('setup_guide_greeting_list_4'),
        intl.getMessage('setup_guide_greeting_list_5'),
    ]);
    return (
        <div class={styles.greeting}>
            <div class={styles.info}>
                <h1 class={styles.title}>{intl.getMessage('setup_guide_greeting_title')}</h1>

                <p class={styles.desc}>{intl.getMessage('setup_guide_greeting_desc')}</p>

                <ul class={styles.list}>
                    <For each={configureList()}>
                        {(item) => <li class={styles.listItem}>{item}</li>}
                    </For>
                </ul>

                <Controls />
            </div>

            <div class={styles.content}>
                <img src={routerImage} alt="Router" />
            </div>
        </div>
    );
};

export default Greeting;
