import { createSignal, createMemo, createEffect } from 'solid-js';
import cn from 'clsx';
import intl from 'panel/common/intl';
import { Dialog } from 'panel/common/ui/Dialog';
import { Button } from 'panel/common/ui/Button';
import { Select } from 'panel/common/controls/Select';
import { Checkbox } from 'panel/common/controls/Checkbox';
import theme from 'panel/lib/theme';

import {
    type DayKey,
    FULL_DAY_END_MS,
    HOURS_OPTIONS,
    MINUTES_OPTIONS,
    type ScheduleDayData,
    getEndTimeOptions,
    getNormalizedEndTime,
    isFullDay,
    msToTime,
    timeToMs,
} from './helpers';
import { getDayName } from './getDayName';
import s from './ScheduleModal.module.pcss';

type Props = {
    visible: boolean;
    currentDay?: DayKey;
    currentData?: ScheduleDayData;
    onClose: () => void;
    onSave: (day: DayKey, start: number, end: number) => void;
};

export const ScheduleModal = (props: Props) => {
    const [startHour, setStartHour] = createSignal(0);
    const [startMinute, setStartMinute] = createSignal(0);
    const [endHour, setEndHour] = createSignal(23);
    const [endMinute, setEndMinute] = createSignal(59);
    const [allDay, setAllDay] = createSignal(false);

    createEffect(() => {
        const data = props.currentData;
        if (!data) {
            setAllDay(false);
            setStartHour(0);
            setStartMinute(0);
            setEndHour(23);
            setEndMinute(59);
            return;
        }

        const isFullDayData = isFullDay(data.start, data.end);
        const start = msToTime(data.start);
        const end = msToTime(data.end);
        const normalizedEnd = getNormalizedEndTime(start, end) ?? end;

        setAllDay(isFullDayData);
        setStartHour(start.hours);
        setStartMinute(start.minutes);
        setEndHour(normalizedEnd.hours);
        setEndMinute(normalizedEnd.minutes);
    });

    const handleAllDayChange = (e: Event) => {
        const checked = (e.target as HTMLInputElement).checked;
        setAllDay(checked);
        if (checked) {
            setStartHour(0);
            setStartMinute(0);
            setEndHour(23);
            setEndMinute(59);
        }
    };

    const applyStartTime = (nextStartHour: number, nextStartMinute: number) => {
        setStartHour(nextStartHour);
        setStartMinute(nextStartMinute);

        if (allDay()) {
            return;
        }

        const normalizedEnd = getNormalizedEndTime(
            { hours: nextStartHour, minutes: nextStartMinute },
            { hours: endHour(), minutes: endMinute() },
        );

        if (!normalizedEnd) {
            return;
        }

        setEndHour(normalizedEnd.hours);
        setEndMinute(normalizedEnd.minutes);
    };

    const applyEndTime = (nextEndHour: number, nextEndMinute: number) => {
        const normalizedEnd = getNormalizedEndTime(
            { hours: startHour(), minutes: startMinute() },
            { hours: nextEndHour, minutes: nextEndMinute },
        );

        if (!normalizedEnd) {
            return;
        }

        setEndHour(normalizedEnd.hours);
        setEndMinute(normalizedEnd.minutes);
    };

    const endTimeOptions = createMemo(() =>
        getEndTimeOptions({ hours: startHour(), minutes: startMinute() }, endHour()),
    );

    const isDisabled = () =>
        !props.currentDay || (!allDay() && !endTimeOptions().hasAvailableEndTime);

    const handleSave = () => {
        if (isDisabled() || !props.currentDay) return;
        const saveStart = allDay() ? 0 : timeToMs(startHour(), startMinute());
        const saveEnd = allDay() ? FULL_DAY_END_MS : timeToMs(endHour(), endMinute());
        props.onSave(props.currentDay, saveStart, saveEnd);
    };

    const dayLabel = () => getDayName(props.currentDay);
    const title = () =>
        props.currentData
            ? intl.getMessage('inactivity_schedule_edit')
            : intl.getMessage('inactivity_schedule_add');

    return (
        <Dialog
            visible={props.visible}
            mask
            onClose={props.onClose}
            title={title()}
            wrapClass="rc-dialog-update"
        >
            <div class={cn(theme.dialog.body, s.content)}>
                <div class={s.dayLabel}>{dayLabel()}</div>

                <div class={s.timeWrapper}>
                    <div class={s.timeField}>
                        <span class={s.timeLabel}>
                            {intl.getMessage('inactivity_schedule_time_from')}
                        </span>
                        <div class={s.selects}>
                            <Select
                                options={HOURS_OPTIONS}
                                value={HOURS_OPTIONS[startHour()]}
                                onChange={(opt) => applyStartTime(opt.value, startMinute())}
                                isDisabled={allDay()}
                                height="big"
                                size="responsive"
                                isSearchable={false}
                            />
                            <Select
                                options={MINUTES_OPTIONS}
                                value={MINUTES_OPTIONS[startMinute()]}
                                onChange={(opt) => applyStartTime(startHour(), opt.value)}
                                isDisabled={allDay()}
                                height="big"
                                size="responsive"
                                isSearchable={false}
                            />
                        </div>
                    </div>
                    <div class={s.timeField}>
                        <span class={s.timeLabel}>
                            {intl.getMessage('inactivity_schedule_time_to')}
                        </span>
                        <div class={s.selects}>
                            <Select
                                options={endTimeOptions().hours}
                                value={HOURS_OPTIONS[endHour()]}
                                onChange={(opt) => applyEndTime(opt.value, endMinute())}
                                isDisabled={allDay()}
                                height="big"
                                size="responsive"
                                isSearchable={false}
                            />
                            <Select
                                options={endTimeOptions().minutes}
                                value={MINUTES_OPTIONS[endMinute()]}
                                onChange={(opt) => applyEndTime(endHour(), opt.value)}
                                isDisabled={allDay()}
                                height="big"
                                size="responsive"
                                isSearchable={false}
                            />
                        </div>
                    </div>
                </div>

                <div class={s.allDayRow}>
                    <Checkbox
                        id="all-day-checkbox"
                        checked={allDay()}
                        onChange={handleAllDayChange}
                    >
                        {intl.getMessage('inactivity_schedule_all_day')}
                    </Checkbox>
                </div>

                <div class={cn(s.notice, theme.text.t2, theme.text.t1_tablet)}>
                    {intl.getMessage('inactivity_schedule_replace_desc')}
                </div>
            </div>

            <div class={theme.dialog.footer}>
                <Button
                    variant="primary"
                    size="small"
                    onClick={handleSave}
                    disabled={isDisabled()}
                    class={theme.dialog.button}
                >
                    {intl.getMessage('save_btn')}
                </Button>
                <Button
                    variant="secondary"
                    size="small"
                    onClick={props.onClose}
                    class={theme.dialog.button}
                >
                    {intl.getMessage('cancel_btn')}
                </Button>
            </div>
        </Dialog>
    );
};
