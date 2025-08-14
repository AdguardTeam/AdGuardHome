import React, { useState } from 'react';

import { ConfirmDialog } from 'panel/common/ui/ConfirmDialog';
import intl from 'panel/common/intl';
import { HOUR } from 'panel/helpers/constants';
import { formatIntervalText } from 'panel/components/Settings/helpers';
import { useDispatch } from 'react-redux';
import { clearLogs, setLogsConfig } from 'panel/actions/queryLogs';
import { Form, FormValues } from './Form';

export type LogsConfigPayload = {
    enabled: boolean;
    anonymize_client_ip: boolean;
    ignore_enabled: boolean;
    ignored: string[];
    interval: number;
};

type Props = {
    interval: number;
    customInterval?: number;
    enabled: boolean;
    anonymize_client_ip: boolean;
    processing: boolean;
    ignored: string[];
    processingClear: boolean;
};

export const LogsConfig = ({
    interval: prevInterval,
    customInterval,
    enabled,
    anonymize_client_ip,
    processing,
    processingClear,
    ignored,
}: Props) => {
    const dispatch = useDispatch();

    const [openConfirmDialog, setOpenConfirmDialog] = useState(false);
    const [confirmConfig, setConfirmConfig] = useState<LogsConfigPayload | null>(null);

    const handleClear = () => {
        setOpenConfirmDialog(true);
    };

    const handleClose = () => {
        setOpenConfirmDialog(false);
    };

    const handleClearConfirm = () => {
        dispatch(clearLogs());
        handleClose();
    };

    const handleFormSubmit = (values: FormValues) => {
        const { interval, customInterval, ...rest } = values;

        const newInterval = customInterval ? customInterval * HOUR : interval;

        const data: LogsConfigPayload = {
            ...rest,
            ignored: values.ignored ? values.ignored.split('\n') : [],
            interval: newInterval,
        };

        if (newInterval < prevInterval) {
            setConfirmConfig(data);
            return;
        }

        dispatch(setLogsConfig(data));
    };

    return (
        <>
            <Form
                initialValues={{
                    enabled,
                    interval: prevInterval,
                    customInterval,
                    anonymize_client_ip,
                    ignored: ignored?.join('\n'),
                }}
                processing={processing}
                processingReset={processingClear}
                onSubmit={handleFormSubmit}
                onReset={handleClear}
            />
            {confirmConfig && (
                <ConfirmDialog
                    onClose={() => setConfirmConfig(null)}
                    onConfirm={() => {
                        dispatch(setLogsConfig(confirmConfig));
                        setConfirmConfig(null);
                    }}
                    buttonText={intl.getMessage('settings_yes_decrease')}
                    cancelText={intl.getMessage('cancel')}
                    title={intl.getMessage('settings_confirm_decrease_log_rotation_interval')}
                    text={intl.getMessage('settings_confirm_decrease_log_rotation_interval_desc', {
                        value: formatIntervalText(confirmConfig.interval),
                    })}
                    buttonVariant="danger"
                />
            )}
            {openConfirmDialog && (
                <ConfirmDialog
                    onClose={handleClose}
                    onConfirm={handleClearConfirm}
                    buttonText={intl.getMessage('settings_yes_clear')}
                    cancelText={intl.getMessage('cancel')}
                    title={intl.getMessage('settings_confirm_clear_query_log')}
                    text={intl.getMessage('settings_confirm_clear_query_log_desc')}
                    buttonVariant="danger"
                />
            )}
        </>
    );
};
