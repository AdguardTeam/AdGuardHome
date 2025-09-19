import React from 'react';
import { useForm } from 'react-hook-form';
import { useDispatch, useSelector } from 'react-redux';

import intl from 'panel/common/intl';
import { Dialog } from 'panel/common/ui/Dialog/Dialog';
import { Button } from 'panel/common/ui/Button';
import { Radio } from 'panel/common/controls/Radio';
import theme from 'panel/lib/theme';
import { setFiltersConfig } from 'panel/actions/filtering';
import { RootState } from 'panel/initialState';
import { ModalWrapper } from 'panel/common/ui/ModalWrapper';
import { MODAL_TYPE } from 'panel/helpers/constants';
import { closeModal } from 'panel/reducers/modals';
import { FilterIntervalInput } from './FilterIntervalInput';

export const FILTER_INTERVALS = {
    DISABLE: 0,
    HOURLY: 1,
    DAILY: 24,
    WEEKLY: 168,
    CUSTOM: -1,
} as const;

const getIntervalTitle = (interval: number) => {
    switch (interval) {
        case FILTER_INTERVALS.DISABLE:
            return intl.getMessage('update_filters_disable');
        case FILTER_INTERVALS.HOURLY:
            return intl.getMessage('update_filters_interval_hourly');
        case FILTER_INTERVALS.DAILY:
            return intl.getMessage('update_filters_interval_daily');
        case FILTER_INTERVALS.WEEKLY:
            return intl.getMessage('update_filters_interval_weekly');
        case FILTER_INTERVALS.CUSTOM:
            return intl.getMessage('update_filters_interval_custom');
        default:
            return intl.getMessage('update_filters_custom_hours', { hours: interval });
    }
};

const RADIO_OPTIONS = [
    { text: getIntervalTitle(FILTER_INTERVALS.DISABLE), value: FILTER_INTERVALS.DISABLE },
    { text: getIntervalTitle(FILTER_INTERVALS.HOURLY), value: FILTER_INTERVALS.HOURLY },
    { text: getIntervalTitle(FILTER_INTERVALS.DAILY), value: FILTER_INTERVALS.DAILY },
    { text: getIntervalTitle(FILTER_INTERVALS.WEEKLY), value: FILTER_INTERVALS.WEEKLY },
    { text: getIntervalTitle(FILTER_INTERVALS.CUSTOM), value: FILTER_INTERVALS.CUSTOM },
];

type FormValues = {
    interval: number;
    customInterval?: number | null;
};

export const FilterUpdateModal = () => {
    const dispatch = useDispatch();
    const { filtering } = useSelector((state: RootState) => state);
    const { processingSetConfig, interval: currentInterval } = filtering;

    const { control, handleSubmit, watch, setValue } = useForm<FormValues>({
        defaultValues: {
            interval: currentInterval || 24,
            customInterval: null,
        },
    });

    const intervalValue = watch('interval');

    const onClose = () => {
        dispatch(closeModal());
    };

    const onSubmit = (data: FormValues) => {
        const finalInterval = data.interval === FILTER_INTERVALS.CUSTOM ? data.customInterval : data.interval;

        if (finalInterval !== null && finalInterval !== undefined) {
            dispatch(
                setFiltersConfig({
                    enabled: filtering.enabled,
                    interval: finalInterval,
                }),
            );
            onClose();
        }
    };

    return (
        <ModalWrapper id={MODAL_TYPE.FILTER_UPDATE}>
            <Dialog visible onClose={onClose} title={intl.getMessage('update_filters_title')}>
                <form onSubmit={handleSubmit(onSubmit)}>
                    <div className={theme.dialog.description}>{intl.getMessage('update_filters_desc')}</div>

                    <div className={theme.form.group}>
                        <Radio
                            name="interval"
                            value={intervalValue}
                            options={RADIO_OPTIONS}
                            handleChange={(value) => setValue('interval', value)}
                            disabled={processingSetConfig}
                        />

                        <FilterIntervalInput
                            control={control}
                            processing={processingSetConfig}
                            intervalValue={intervalValue}
                        />
                    </div>

                    <div className={theme.dialog.footer}>
                        <Button
                            type="submit"
                            variant="primary"
                            size="small"
                            disabled={processingSetConfig}
                            className={theme.dialog.button}
                        >
                            {intl.getMessage('save')}
                        </Button>

                        <Button
                            type="button"
                            variant="secondary"
                            size="small"
                            onClick={onClose}
                            className={theme.dialog.button}
                        >
                            {intl.getMessage('cancel')}
                        </Button>
                    </div>
                </form>
            </Dialog>
        </ModalWrapper>
    );
};
