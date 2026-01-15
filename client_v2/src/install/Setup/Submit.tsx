import React from 'react';
import { useSelector } from 'react-redux';

import intl from 'panel/common/intl';
import { Button } from 'panel/common/ui/Button';
import { RootState } from 'panel/initialState';
import Controls from './Controls';
import { WebConfig } from './Settings';
import setup from './styles.module.pcss'

type Props = {
    webConfig: WebConfig;
    openDashboard: (ip: string, port: number) => void;
    onSubmit: () => void;
};

export const Submit = ({ openDashboard, webConfig, onSubmit }: Props) => {
    const { processingSubmit, submitted } = useSelector((state: RootState) => state.install!);

    return (
        <div className={setup.end}>
            <div className={setup.group}>
                <h1 className={setup.titleStep}>{intl.getMessage('install_submit_title')}</h1>

                <p className={setup.desc}>
                    {intl.getMessage('setup_complete')}
                </p>
            </div>

            {!submitted ? (
                <Button
                    id="install_save"
                    type="button"
                    size="small"
                    variant="primary"
                    className={setup.button}
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
