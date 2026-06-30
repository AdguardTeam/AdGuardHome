import { createSignal, createMemo, createEffect, untrack } from 'solid-js';

import intl from 'panel/common/intl';
import { Dialog } from 'panel/common/ui/Dialog/Dialog';
import { Button } from 'panel/common/ui/Button';
import { Radio } from 'panel/common/controls/Radio';
import theme from 'panel/lib/theme';
import { setFiltersConfig, filteringState } from 'panel/stores/filtering';
import { ModalWrapper } from 'panel/common/ui/ModalWrapper';
import { MODAL_TYPE } from 'panel/helpers/constants';
import { closeModal } from 'panel/stores/modals';
import { FilterIntervalInput, FILTER_INTERVAL_RANGE } from './FilterIntervalInput';

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

export const FilterUpdateModal = () => {
    const PREDEFINED_INTERVALS: number[] = [
        FILTER_INTERVALS.DISABLE,
        FILTER_INTERVALS.HOURLY,
        FILTER_INTERVALS.DAILY,
        FILTER_INTERVALS.WEEKLY,
    ];

    const isCustom = createMemo(() => {
        const currentInterval = filteringState.interval;
        return currentInterval != null && !PREDEFINED_INTERVALS.includes(currentInterval);
    });

    const [interval, setIntervalValue] = createSignal<number>(
        untrack(() => (isCustom() ? FILTER_INTERVALS.CUSTOM : (filteringState.interval ?? 24))),
    );
    const [customInterval, setCustomInterval] = createSignal<number | null>(
        untrack(() => (isCustom() ? filteringState.interval : null)),
    );

    createEffect(() => {
        const currentInterval = filteringState.interval;
        const custom = currentInterval != null && !PREDEFINED_INTERVALS.includes(currentInterval);
        setIntervalValue(custom ? FILTER_INTERVALS.CUSTOM : (currentInterval ?? 24));
        setCustomInterval(custom ? currentInterval : null);
    });

    const intervalValue = createMemo(() => interval());

    const onClose = () => {
        const currentInterval = filteringState.interval;
        const custom = currentInterval != null && !PREDEFINED_INTERVALS.includes(currentInterval);
        setIntervalValue(custom ? FILTER_INTERVALS.CUSTOM : (currentInterval ?? 24));
        setCustomInterval(custom ? currentInterval : null);
        closeModal();
    };

    const onSubmit = (e: Event) => {
        e.preventDefault();
        const finalInterval =
            interval() === FILTER_INTERVALS.CUSTOM ? customInterval() : interval();

        if (finalInterval === null || finalInterval === undefined || finalInterval < 0) {
            return;
        }

        if (interval() === FILTER_INTERVALS.CUSTOM) {
            if (
                finalInterval < FILTER_INTERVAL_RANGE.MIN ||
                finalInterval > FILTER_INTERVAL_RANGE.MAX
            ) {
                return;
            }
        }

        setFiltersConfig({
            enabled: filteringState.enabled,
            interval: finalInterval,
        });
        onClose();
    };

    return (
        <ModalWrapper id={MODAL_TYPE.FILTER_UPDATE}>
            <Dialog visible onClose={onClose} title={intl.getMessage('update_filters_title')}>
                <form onSubmit={onSubmit}>
                    <div class={theme.dialog.description}>
                        {intl.getMessage('update_filters_desc')}
                    </div>

                    <div class={theme.form.group}>
                        <Radio
                            name="interval"
                            value={intervalValue()}
                            options={RADIO_OPTIONS}
                            handleChange={(value: number) => setIntervalValue(value)}
                            disabled={filteringState.processingSetConfig}
                        />

                        <FilterIntervalInput
                            value={customInterval()}
                            onChange={(value) => setCustomInterval(value)}
                            processing={filteringState.processingSetConfig}
                            intervalValue={intervalValue()}
                        />
                    </div>

                    <div class={theme.dialog.footer}>
                        <Button
                            type="submit"
                            variant="primary"
                            size="small"
                            disabled={filteringState.processingSetConfig}
                            class={theme.dialog.button}
                        >
                            {intl.getMessage('save')}
                        </Button>

                        <Button
                            type="button"
                            variant="secondary"
                            size="small"
                            onClick={onClose}
                            class={theme.dialog.button}
                        >
                            {intl.getMessage('cancel')}
                        </Button>
                    </div>
                </form>
            </Dialog>
        </ModalWrapper>
    );
};
