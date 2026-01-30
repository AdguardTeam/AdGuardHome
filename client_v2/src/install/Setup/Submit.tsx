import React, { useEffect } from 'react';
import { useSelector } from 'react-redux';

import intl from 'panel/common/intl';
import { Button } from 'panel/common/ui/Button';
import { RootState } from 'panel/initialState';
import Controls from './Controls';
import type { WebConfig } from './types';
import styles from './styles.module.pcss';

type Props = {
    webConfig: WebConfig;
    openDashboard: (ip: string, port: number) => void;
    onSubmit: () => void;
};

export const Submit = ({ openDashboard, webConfig, onSubmit }: Props) => {
    const { processingSubmit, submitted } = useSelector((state: RootState) => state.install!);

    useEffect(() => {
        if (submitted) {
            openDashboard(webConfig.ip, webConfig.port);
        }
    }, [submitted, openDashboard, webConfig.ip, webConfig.port]);

    return (
        <div className={styles.end}>
            <div className={styles.group}>
                <h1 className={styles.titleStep}>{intl.getMessage('install_submit_title')}</h1>

                <p className={styles.desc}>
                    {intl.getMessage('setup_complete')}
                </p>
            </div>

            {!submitted ? (
                <Button
                    id="install_save"
                    type="button"
                    size="small"
                    variant="primary"
                    className={styles.button}
                    disabled={processingSubmit}
                    onClick={onSubmit}
                >
                    {intl.getMessage('open_dashboard')}
                </Button>
            ) : (
                <Controls openDashboard={openDashboard} ip={webConfig.ip} port={webConfig.port} />
            )}
        </div>
    );
};
