import { createSignal, Show } from 'solid-js';

import { ConfirmDialog } from 'panel/common/ui/ConfirmDialog';
import intl from 'panel/common/intl';
import { HOUR } from 'panel/helpers/constants';
import { formatIntervalText } from 'panel/components/Settings/helpers';
import { clearLogs, setLogsConfig } from 'panel/stores/queryLogs';
import { Form, type FormValues } from './Form';

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

export const LogsConfig = (props: Props) => {
    const [openConfirmDialog, setOpenConfirmDialog] = createSignal(false);
    const [confirmConfig, setConfirmConfig] = createSignal<LogsConfigPayload | null>(null);

    const handleClear = () => {
        setOpenConfirmDialog(true);
    };

    const handleClose = () => {
        setOpenConfirmDialog(false);
    };

    const handleClearConfirm = () => {
        clearLogs();
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

        if (newInterval < props.interval) {
            setConfirmConfig(data);
            return;
        }

        setLogsConfig(data);
    };

    return (
        <>
            <Form
                initialValues={{
                    enabled: props.enabled,
                    interval: props.interval,
                    customInterval: props.customInterval,
                    anonymize_client_ip: props.anonymize_client_ip,
                    ignored: props.ignored?.join('\n'),
                }}
                processing={props.processing}
                processingReset={props.processingClear}
                onSubmit={handleFormSubmit}
                onReset={handleClear}
            />
            <Show when={confirmConfig()}>
                {(config) => (
                    <ConfirmDialog
                        onClose={() => setConfirmConfig(null)}
                        onConfirm={() => {
                            setLogsConfig(config());
                            setConfirmConfig(null);
                        }}
                        buttonText={intl.getMessage('settings_yes_decrease')}
                        cancelText={intl.getMessage('cancel')}
                        title={intl.getMessage('settings_confirm_decrease_log_rotation_interval')}
                        text={intl.getMessage(
                            'settings_confirm_decrease_log_rotation_interval_desc',
                            {
                                value: formatIntervalText(config().interval),
                            },
                        )}
                        buttonVariant="danger"
                    />
                )}
            </Show>
            <Show when={openConfirmDialog()}>
                <ConfirmDialog
                    onClose={handleClose}
                    onConfirm={handleClearConfirm}
                    buttonText={intl.getMessage('settings_yes_clear')}
                    cancelText={intl.getMessage('cancel')}
                    title={intl.getMessage('settings_confirm_clear_query_log')}
                    text={intl.getMessage('settings_confirm_clear_query_log_desc')}
                    buttonVariant="danger"
                />
            </Show>
        </>
    );
};
