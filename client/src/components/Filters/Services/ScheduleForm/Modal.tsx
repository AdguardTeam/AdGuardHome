import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';

import ReactModal from 'react-modal';

import { Timezone } from './Timezone';

import { TimeSelect } from './TimeSelect';

import { TimePeriod } from './TimePeriod';
import { getFullDayName, getShortDayName } from './helpers';
import { LOCAL_TIMEZONE_VALUE } from '../../../../helpers/constants';

export const DAYS_OF_WEEK = ['sun', 'mon', 'tue', 'wed', 'thu', 'fri', 'sat'];

const INITIAL_START_TIME_MS = 0;
const INITIAL_END_TIME_MS = 86340000;

interface ModalProps {
    schedule: {
        time_zone: string;
    };
    currentDay?: string;
    isOpen: boolean;
    onClose: (...args: unknown[]) => unknown;
    onSubmit: (values: any) => void;
}

export const Modal = ({ isOpen, currentDay, schedule, onClose, onSubmit }: ModalProps) => {
    const [t] = useTranslation();

    const intialTimezone =
        schedule.time_zone === LOCAL_TIMEZONE_VALUE
            ? Intl.DateTimeFormat().resolvedOptions().timeZone
            : schedule.time_zone;

    const [timezone, setTimezone] = useState(intialTimezone);
    const [days, setDays] = useState<Set<string>>(new Set());

    const [startTime, setStartTime] = useState(INITIAL_START_TIME_MS);
    const [endTime, setEndTime] = useState(INITIAL_END_TIME_MS);

    const [wrongPeriod, setWrongPeriod] = useState(true);

    useEffect(() => {
        if (currentDay) {
            const newDays = new Set([currentDay]);
            setDays(newDays);

            setStartTime(schedule[currentDay].start);
            setEndTime(schedule[currentDay].end);
        }
    }, [currentDay]);

    useEffect(() => {
        if (startTime >= endTime) {
            setWrongPeriod(true);
        } else {
            setWrongPeriod(false);
        }
    }, [startTime, endTime]);

    const addDays = (day: any) => {
        const newDays = new Set(days);

        if (newDays.has(day)) {
            newDays.delete(day);
        } else {
            newDays.add(day);
        }

        setDays(newDays);
    };

    const activeDay = (day: any) => {
        return days.has(day);
    };

    const onFormSubmit = (e: any) => {
        e.preventDefault();

        const newSchedule = schedule;

        Array.from(days).forEach((day) => {
            newSchedule[day] = {
                start: startTime,
                end: endTime,
            };
        });

        if (timezone !== intialTimezone) {
            newSchedule.time_zone = timezone;
        }

        onSubmit(newSchedule);
    };

    return (
        <ReactModal
            className="Modal__Bootstrap modal-dialog modal-dialog-centered modal-dialog--schedule"
            closeTimeoutMS={0}
            isOpen={isOpen}
            onRequestClose={onClose}>
            <div className="modal-content">
                <div className="modal-header">
                    <h4 className="modal-title">{currentDay ? t('schedule_edit') : t('schedule_new')}</h4>

                    <button type="button" className="close" onClick={onClose}>
                        <span className="sr-only">Close</span>
                    </button>
                </div>

                <form onSubmit={onFormSubmit}>
                    <div className="modal-body">
                        <Timezone timezone={timezone} setTimezone={setTimezone} />

                        <div className="schedule__days">
                            {DAYS_OF_WEEK.map((day) => (
                                <button
                                    type="button"
                                    key={day}
                                    className="btn schedule__button-day"
                                    data-active={activeDay(day)}
                                    onClick={() => addDays(day)}>
                                    {getShortDayName(t, day)}
                                </button>
                            ))}
                        </div>

                        <div className="schedule__time-wrap">
                            <div className="schedule__time-row">
                                <TimeSelect value={startTime} onChange={(v) => setStartTime(v)} />

                                <TimeSelect value={endTime} onChange={(v) => setEndTime(v)} />
                            </div>

                            {wrongPeriod && <div className="schedule__error">{t('schedule_invalid_select')}</div>}
                        </div>

                        <div className="schedule__info">
                            <div className="schedule__info-title">{t('schedule_modal_time_off')}</div>

                            <div className="schedule__info-row">
                                <svg className="icons schedule__info-icon">
                                    <use xlinkHref="#calendar" />
                                </svg>
                                {days.size ? (
                                    Array.from(days)
                                        .map((day) => getFullDayName(t, day))
                                        .join(', ')
                                ) : (
                                    <span>—</span>
                                )}
                            </div>

                            <div className="schedule__info-row">
                                <svg className="icons schedule__info-icon">
                                    <use xlinkHref="#watch" />
                                </svg>
                                {wrongPeriod ? (
                                    <span>—</span>
                                ) : (
                                    <TimePeriod startTimeMs={startTime} endTimeMs={endTime} />
                                )}
                            </div>
                        </div>

                        <div className="schedule__notice">{t('schedule_modal_description')}</div>
                    </div>

                    <div className="modal-footer">
                        <div className="btn-list">
                            <button
                                type="button"
                                className="btn btn-success btn-standard"
                                disabled={days.size === 0 || wrongPeriod}
                                onClick={onFormSubmit}>
                                {currentDay ? t('schedule_save') : t('schedule_add')}
                            </button>
                        </div>
                    </div>
                </form>
            </div>
        </ReactModal>
    );
};
