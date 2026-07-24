import { For, createMemo } from 'solid-js';
import cn from 'clsx';
import intl from 'panel/common/intl';
import { Select } from 'panel/common/controls/Select';
import { LANGUAGES } from 'panel/helpers/twosky';
import { installState, setLanguage } from 'panel/stores/install';
import { Controls } from './Controls';
import styles from './styles.module.pcss';

import routerImage from '../../img/router.svg';
import type { Lang } from 'panel/api/model/lang';
import theme from 'panel/lib/theme';

export const Greeting = () => {
    const languageOptions = () =>
        Object.entries(LANGUAGES).map(([value, label]) => ({
            value,
            label,
        }));

    const configureList = createMemo(() => [
        intl.getMessage('setup_guide_greeting_list_1'),
        intl.getMessage('setup_guide_greeting_list_2'),
        intl.getMessage('setup_guide_greeting_list_3'),
        intl.getMessage('setup_guide_greeting_list_4'),
        intl.getMessage('setup_guide_greeting_list_5'),
    ]);

    const selectedLanguage = () =>
        languageOptions().find((opt) => opt.value === installState.language) ||
        languageOptions()[0];

    return (
        <div class={styles.greeting}>
            <div class={styles.info}>
                <h1 class={styles.title}>{intl.getMessage('setup_guide_greeting_title')}</h1>

                <label class={cn(styles.langLabel, theme.text.t3)}>
                    {intl.getMessage('select_language')}
                </label>

                <Select
                    options={languageOptions()}
                    value={selectedLanguage()}
                    onChange={(option) => setLanguage(option.value as Lang)}
                    height="big"
                    menuSize="big"
                    class={styles.langSelect}
                    size="responsive"
                />

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
