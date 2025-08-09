import React, { useState } from 'react';

import { ConfirmDialog } from 'panel/common/ui/ConfirmDialog';
import intl from 'panel/common/intl';
import { HOUR } from 'panel/helpers/constants';
import { formatIntervalText } from 'panel/components/Settings/helpers';

import { Form, FormValues } from './Form';

export type StatsConfigPayload = {
    enabled: boolean;
    ignored: string[];
    interval: number;
};

type Props = {
    interval: number;
    customInterval?: number;
    ignored: string[];
    enabled: boolean;
    processing: boolean;
    processingReset: boolean;
    setStatsConfig: (config: StatsConfigPayload) => void;
    resetStats: () => void;
};

export const StatsConfig = ({
    interval: prevInterval,
    customInterval,
    ignored,
    enabled,
    processing,
    processingReset,
    setStatsConfig,
    resetStats,
}: Props) => {
    const [openClearDialog, setOpenClearDialog] = useState(false);
    const [confirmConfig, setConfirmConfig] = useState<StatsConfigPayload | null>(null);

    const handleClear = () => {
        setOpenClearDialog(true);
    };

    const handleClose = () => {
        setOpenClearDialog(false);
    };

    const handleClearConfirm = () => {
        resetStats();
        handleClose();
    };

    const handleFormSubmit = (values: FormValues) => {
        const { interval, customInterval, enabled, ignored } = values;

        const newInterval = customInterval ? customInterval * HOUR : interval;

        const data: StatsConfigPayload = {
            enabled,
            interval: newInterval,
            ignored: ignored ? ignored.split('\n') : [],
        };

        if (newInterval < prevInterval) {
            setConfirmConfig(data);
            return;
        }

        setStatsConfig(data);
    };

    return (
        <>
            <Form
                initialValues={{
                    interval: prevInterval,
                    customInterval,
                    enabled,
                    ignored: ignored.join('\n'),
                    ignore_enabled: false,
                }}
                processing={processing}
                processingReset={processingReset}
                onSubmit={handleFormSubmit}
                onReset={handleClear}
            />
            {confirmConfig && (
                <ConfirmDialog
                    onClose={() => setConfirmConfig(null)}
                    onConfirm={() => {
                        setStatsConfig(confirmConfig);
                        setConfirmConfig(null);
                    }}
                    buttonText={intl.getMessage('settings_yes_decrease')}
                    cancelText={intl.getMessage('cancel')}
                    title={intl.getMessage('settings_confirm_decrease_stats_rotation_interval')}
                    text={intl.getMessage('settings_confirm_decrease_stats_rotation_interval_desc', {
                        value: formatIntervalText(confirmConfig.interval),
                    })}
                    buttonVariant="danger"
                />
            )}
            {openClearDialog && (
                <ConfirmDialog
                    onClose={handleClose}
                    onConfirm={handleClearConfirm}
                    buttonText={intl.getMessage('settings_yes_clear')}
                    cancelText={intl.getMessage('cancel')}
                    title={intl.getMessage('settings_confirm_clear_statistics')}
                    text={intl.getMessage('settings_confirm_clear_statistics_desc')}
                    buttonVariant="danger"
                />
            )}
        </>
    );
};
