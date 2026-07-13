import { createEffect, Show } from 'solid-js';

import intl from 'panel/common/intl';
import { Button } from 'panel/common/ui/Button';
import { installState } from 'panel/stores/install';
import { Controls } from './Controls';
import type { WebConfig } from './types';
import styles from './styles.module.pcss';

type Props = {
    webConfig: WebConfig;
    openDashboard: (ip: string, port: number) => void;
    onSubmit: () => void;
};

export const Submit = (props: Props) => {
    createEffect(() => {
        if (installState.submitted) {
            props.openDashboard(props.webConfig.ip, props.webConfig.port);
        }
    });

    return (
        <div class={styles.end}>
            <div class={styles.group}>
                <h1 class={styles.titleStep}>{intl.getMessage('install_submit_title')}</h1>

                <p class={styles.desc}>{intl.getMessage('setup_complete')}</p>
            </div>

            <Show
                when={!installState.submitted}
                fallback={
                    <Controls
                        openDashboard={props.openDashboard}
                        ip={props.webConfig.ip}
                        port={props.webConfig.port}
                    />
                }
            >
                <Button
                    id="install_save"
                    type="button"
                    size="small"
                    variant="primary"
                    class={styles.button}
                    disabled={installState.processingSubmit}
                    onClick={props.onSubmit}
                >
                    {intl.getMessage('open_dashboard')}
                </Button>
            </Show>
        </div>
    );
};
