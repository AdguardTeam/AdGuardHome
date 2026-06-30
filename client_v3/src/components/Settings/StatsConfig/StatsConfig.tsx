import { createSignal, Show } from 'solid-js';

import { ConfirmDialog } from 'panel/common/ui/ConfirmDialog';
import intl from 'panel/common/intl';
import { HOUR } from 'panel/helpers/constants';
import { formatIntervalText } from 'panel/components/Settings/helpers';

import { resetStats, setStatsConfig } from 'panel/stores/stats';
import { Form, type FormValues } from './Form';

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
};

export const StatsConfig = (props: Props) => {
    const [openClearDialog, setOpenClearDialog] = createSignal(false);
    const [confirmConfig, setConfirmConfig] = createSignal<StatsConfigPayload | null>(null);

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

        let newInterval: number;
        if (customInterval) {
            newInterval = customInterval >= HOUR ? customInterval : customInterval * HOUR;
        } else {
            newInterval = interval;
        }

        const data: StatsConfigPayload = {
            enabled,
            interval: newInterval,
            ignored: ignored ? ignored.split('\n') : [],
        };

        if (newInterval < props.interval) {
            setConfirmConfig(data);
            return;
        }

        setStatsConfig(data);
    };

    return (
        <>
            <Form
                initialValues={{
                    interval: props.interval,
                    customInterval: props.customInterval,
                    enabled: props.enabled,
                    ignored: props.ignored.join('\n'),
                    ignore_enabled: false,
                }}
                processing={props.processing}
                processingReset={props.processingReset}
                onSubmit={handleFormSubmit}
                onReset={handleClear}
            />
            <Show when={confirmConfig()}>
                {(config) => (
                    <ConfirmDialog
                        onClose={() => setConfirmConfig(null)}
                        onConfirm={() => {
                            setStatsConfig(config());
                            setConfirmConfig(null);
                        }}
                        buttonText={intl.getMessage('settings_yes_decrease')}
                        cancelText={intl.getMessage('cancel')}
                        title={intl.getMessage('settings_confirm_decrease_stats_rotation_interval')}
                        text={intl.getMessage(
                            'settings_confirm_decrease_stats_rotation_interval_desc',
                            {
                                value: formatIntervalText(config().interval),
                            },
                        )}
                        buttonVariant="danger"
                    />
                )}
            </Show>
            <Show when={openClearDialog()}>
                <ConfirmDialog
                    onClose={handleClose}
                    onConfirm={handleClearConfirm}
                    buttonText={intl.getMessage('settings_yes_clear')}
                    cancelText={intl.getMessage('cancel')}
                    title={intl.getMessage('settings_confirm_clear_statistics')}
                    text={intl.getMessage('settings_confirm_clear_statistics_desc')}
                    buttonVariant="danger"
                />
            </Show>
        </>
    );
};
